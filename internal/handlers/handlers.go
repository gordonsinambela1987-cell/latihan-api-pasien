package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Patient merepresentasikan struktur data untuk seorang pasien.
// DateOfBirth adalah string agar sesuai dengan input/output JSON.
type Patient struct {
	ID          int       `json:"id"`
	KTPNumber   string    `json:"ktpNumber"`
	FullName    string    `json:"fullName"`
	DateOfBirth string    `json:"dateOfBirth"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Doctor merepresentasikan struktur data untuk seorang dokter.
type Doctor struct {
	ID        int    `json:"id"`
	NIK       string `json:"nik"`
	Name      string `json:"name"`
	Specialty string `json:"specialty"`
}

// Appointment merepresentasikan struktur data untuk janji temu.
type Appointment struct {
	ID              int       `json:"id"`
	PatientID       int       `json:"patientId"`
	DoctorID        int       `json:"doctorId"`
	AppointmentDate time.Time `json:"appointmentDate"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
}

// AppointmentResponse adalah struktur data yang akan dikirim sebagai JSON.
type AppointmentResponse struct {
	ID              int       `json:"id"`
	DoctorID        int       `json:"doctorId"`
	DoctorName      string    `json:"doctorName"`
	AppointmentDate time.Time `json:"appointmentDate"`
	Status          string    `json:"status"`
}

// RescheduleRequest adalah struktur data untuk body JSON saat reschedule.
type RescheduleRequest struct {
	NewAppointmentDate time.Time `json:"newAppointmentDate"`
}

// ScheduleRequest Dokter adalah struktur untuk body JSON saat menambah jadwal.
type ScheduleRequest struct {
	DayOfWeek int    `json:"dayOfWeek"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

// ScheduleResponse adalah struktur untuk menampilkan jadwal dokter.
type ScheduleResponse struct {
	DayOfWeek int    `json:"dayOfWeek"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

// TimeOffRequest adalah struktur untuk body JSON saat menambah hari libur.
type TimeOffRequest struct {
	OffDate string `json:"offDate"`          // Format: YYYY-MM-DD
	Reason  string `json:"reason,omitempty"` // omitempty berarti field ini opsional
}

// CreatePatientHandler menangani pembuatan pasien baru.
func CreatePatientHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p Patient
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			log.Printf("Error decoding JSON body: %v", err)
			http.Error(w, "Request body tidak valid", http.StatusBadRequest)
			return
		}

		// Validasi input
		if len(p.KTPNumber) != 16 {
			http.Error(w, "Nomor KTP harus 16 digit", http.StatusBadRequest)
			return
		}
		if match, _ := regexp.MatchString("^[0-9]+$", p.KTPNumber); !match {
			http.Error(w, "Nomor KTP harus berupa angka.", http.StatusBadRequest)
			return
		}
		if len(p.FullName) < 3 {
			http.Error(w, "Nama lengkap minimal 3 karakter", http.StatusBadRequest)
			return
		}

		// Validasi DAN KONVERSI format tanggal (ini diperlukan)
		layout := "02-01-2006" // Format DD-MM-YYYY
		dob, err := time.Parse(layout, p.DateOfBirth)
		if err != nil {
			http.Error(w, "Format tanggal lahir harus DD-MM-YYYY", http.StatusBadRequest)
			return
		}
		// --- VALIDASI BARU: Cek apakah tanggal lahir ada di masa depan ---
		if dob.After(time.Now()) {
			http.Error(w, "Tanggal lahir tidak boleh ada di masa depan.", http.StatusBadRequest)
			return
		}

		// Masukkan data ke database menggunakan tanggal yang sudah dikonversi
		query := `INSERT INTO patients (ktp_number, full_name, date_of_birth) 
                  VALUES ($1, $2, $3) 
                  RETURNING id, created_at`

		err = dbpool.QueryRow(context.Background(), query, p.KTPNumber, p.FullName, dob).Scan(&p.ID, &p.CreatedAt)
		if err != nil {
			// Cek apakah error ini adalah error 'unique violation' dari Postgres
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 adalah kode untuk unique_violation
				http.Error(w, "Pasien dengan nomor KTP tersebut sudah terdaftar.", http.StatusConflict) // Kirim 409 Conflict
				return
			}
			log.Printf("Gagal memasukkan pasien ke DB: %v", err)
			http.Error(w, "Gagal menyimpan data pasien", http.StatusInternalServerError)
			return
		}

		// Kirim response JSON yang sukses
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)
	}
}

// GetPatientByIDHandler adalah fungsi untuk mengambil satu pasien berdasarkan ID.
func GetPatientByIDHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var p Patient
		var dob time.Time // Variabel sementara untuk menampung tanggal dari DB
		query := `SELECT id, ktp_number, full_name, date_of_birth, created_at 
                  FROM patients 
                  WHERE id = $1`

		err := dbpool.QueryRow(context.Background(), query, id).Scan(&p.ID, &p.KTPNumber, &p.FullName, &dob, &p.CreatedAt)
		if err != nil {
			if err.Error() == "no rows in result set" {
				http.Error(w, "Pasien tidak ditemukan", http.StatusNotFound)
				return
			}
			http.Error(w, "Gagal mengambil data pasien", http.StatusInternalServerError)
			return
		}

		// Konversi tanggal dari DB ke format DD-MM-YYYY untuk response
		layout := "02-01-2006"
		p.DateOfBirth = dob.Format(layout)

		// Kirim response JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	}
}

// CreateDoctorHandler adalah fungsi untuk mendaftarkan dokter baru.
func CreateDoctorHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Dekode request JSON ke dalam struct Doctor
		var d Doctor
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			log.Printf("Error decoding JSON body: %v", err)
			http.Error(w, "Request body tidak valid", http.StatusBadRequest)
			return
		}

		// 2. Validasi input
		if len(d.NIK) != 10 {
			http.Error(w, "NIK dokter harus 10 digit", http.StatusBadRequest)
			return
		}
		if match, _ := regexp.MatchString("^[0-9]+$", d.NIK); !match {
			http.Error(w, "NIK harus berupa angka.", http.StatusBadRequest)
		}
		if len(d.Name) < 3 {
			http.Error(w, "Nama dokter minimal 3 karakter", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(d.Specialty) == "" {
			http.Error(w, "Specialty tidak boleh kosong.", http.StatusBadRequest)
			return
		}

		// 3. Masukkan data ke database
		query := `INSERT INTO doctors (nik, name, specialty) 
                  VALUES ($1, $2, $3) 
                  RETURNING id`

		err := dbpool.QueryRow(context.Background(), query, d.NIK, d.Name, d.Specialty).Scan(&d.ID)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				http.Error(w, "Dokter dengan NIK tersebut sudah terdaftar.", http.StatusConflict) // Kirim 409
				return
			}
			// (Nanti kita bisa tambahkan pengecekan NIK duplikat di sini)
			log.Printf("Gagal memasukkan dokter ke DB: %v", err)
			http.Error(w, "Gagal menyimpan data dokter", http.StatusInternalServerError)
			return
		}

		// 4. Kirim response JSON yang sukses
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // Status 201 Created
		json.NewEncoder(w).Encode(d)
	}
}

// GetAllDoctorsHandler adalah fungsi untuk mengambil semua data dokter.
func GetAllDoctorsHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Siapkan query untuk mengambil semua dokter
		query := `SELECT id, nik, name, specialty FROM doctors`

		rows, err := dbpool.Query(context.Background(), query)
		if err != nil {
			http.Error(w, "Gagal mengambil data dokter", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// 2. Looping melalui hasil query dan masukkan ke dalam slice
		var doctors []Doctor
		for rows.Next() {
			var d Doctor
			if err := rows.Scan(&d.ID, &d.NIK, &d.Name, &d.Specialty); err != nil {
				http.Error(w, "Gagal memindai data dokter", http.StatusInternalServerError)
				return
			}
			doctors = append(doctors, d)
		}

		// Jika tidak ada dokter sama sekali, kembalikan array kosong, bukan error
		if doctors == nil {
			doctors = []Doctor{}
		}

		// 3. Kirim response JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doctors)
	}
}

// CreateAppointmentHandler menangani pembuatan janji temu baru.
func CreateAppointmentHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Dekode request JSON
		var appt Appointment
		if err := json.NewDecoder(r.Body).Decode(&appt); err != nil {
			http.Error(w, "Request body tidak valid", http.StatusBadRequest)
			return
		}

		// --- LOGIKA VALIDASI JADWAL (DITAMBAHKAN DI SINI) ---
		apptDate := appt.AppointmentDate
		doctorID := appt.DoctorID

		// 2. Pengecekan #1: Apakah dokter libur pada tanggal tersebut?
		var count int
		err := dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM doctor_time_off WHERE doctor_id = $1 AND off_date = $2", doctorID, apptDate.Format("2006-01-02")).Scan(&count)
		if err != nil || count > 0 {
			http.Error(w, "Dokter tidak tersedia pada tanggal tersebut (libur).", http.StatusConflict) // 409 Conflict
			return
		}

		// 3. Pengecekan #2: Apakah sesuai dengan jadwal kerja mingguan?
		dayOfWeek := int(apptDate.Weekday())
		if dayOfWeek == 0 {
			dayOfWeek = 7
		} // Konversi Minggu dari 0 ke 7

		var startTime, endTime string
		err = dbpool.QueryRow(context.Background(), "SELECT start_time, end_time FROM doctor_schedules WHERE doctor_id = $1 AND day_of_week = $2", doctorID, dayOfWeek).Scan(&startTime, &endTime)

		requestTime := apptDate.Format("15:04:05")
		if err != nil || requestTime < startTime || requestTime > endTime {
			http.Error(w, "Jadwal yang diminta di luar jam kerja dokter.", http.StatusConflict)
			return
		}

		// 4. Pengecekan #3: Apakah bentrok dengan janji temu lain?
		err = dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM appointments WHERE doctor_id = $1 AND appointment_date = $2", doctorID, apptDate).Scan(&count)
		if err != nil || count > 0 {
			http.Error(w, "Slot waktu yang diminta sudah terisi. Silakan pilih jam lain.", http.StatusConflict)
			return
		}
		// --- AKHIR VALIDASI JADWAL ---

		// 5. Jika lolos, masukkan data ke database
		query := `INSERT INTO appointments (patient_id, doctor_id, appointment_date) 
                  VALUES ($1, $2, $3) 
                  RETURNING id, status, created_at`

		err = dbpool.QueryRow(context.Background(), query, appt.PatientID, appt.DoctorID, appt.AppointmentDate).Scan(&appt.ID, &appt.Status, &appt.CreatedAt)
		if err != nil {
			// (Penanganan foreign key error)
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
				http.Error(w, "Patient atau Doctor dengan ID tersebut tidak ditemukan.", http.StatusNotFound)
				return
			}
			log.Printf("Gagal menyimpan janji temu: %v", err)
			http.Error(w, "Gagal menyimpan janji temu", http.StatusInternalServerError)
			return
		}

		// 6. Kirim response JSON yang sukses
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(appt)
	}
}

// GetAppointmentsByPatientIDHandler mengambil semua janji temu milik satu pasien.
func GetAppointmentsByPatientIDHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Ambil ID pasien dari URL
		patientID := r.PathValue("id")

		// 2. Query ke database dengan JOIN untuk mendapatkan nama dokter
		query := `
            SELECT a.id, a.doctor_id, d.name, a.appointment_date, a.status
            FROM appointments a
            JOIN doctors d ON a.doctor_id = d.id
            WHERE a.patient_id = $1
            ORDER BY a.appointment_date DESC`

		rows, err := dbpool.Query(context.Background(), query, patientID)
		if err != nil {
			http.Error(w, "Gagal mengambil data janji temu", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// 3. Looping melalui hasil dan masukkan ke dalam slice
		var appointments []AppointmentResponse
		for rows.Next() {
			var appt AppointmentResponse
			if err := rows.Scan(&appt.ID, &appt.DoctorID, &appt.DoctorName, &appt.AppointmentDate, &appt.Status); err != nil {
				http.Error(w, "Gagal memindai data janji temu", http.StatusInternalServerError)
				return
			}
			appointments = append(appointments, appt)
		}

		if appointments == nil {
			appointments = []AppointmentResponse{}
		}

		// 4. Kirim response JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appointments)
	}
}

// RescheduleAppointmentHandler menangani penjadwalan ulang janji temu.
func RescheduleAppointmentHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Ambil ID janji temu dari URL
		appointmentID := r.PathValue("id")

		// 2. Dekode body JSON
		var req RescheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Request body tidak valid", http.StatusBadRequest)
			return
		}

		// 3. Ambil DoctorID dari janji temu yang ada
		var doctorID int
		err := dbpool.QueryRow(context.Background(), "SELECT doctor_id FROM appointments WHERE id = $1", appointmentID).Scan(&doctorID)
		if err != nil {
			if err.Error() == "no rows in result set" {
				http.Error(w, "Janji temu tidak ditemukan", http.StatusNotFound)
				return
			}
			http.Error(w, "Gagal mengambil data janji temu", http.StatusInternalServerError)
			return
		}

		// --- LOGIKA VALIDASI JADWAL ---
		newDate := req.NewAppointmentDate

		// 4. Pengecekan #1: Apakah dokter libur pada tanggal tersebut?
		var count int
		err = dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM doctor_time_off WHERE doctor_id = $1 AND off_date = $2", doctorID, newDate.Format("2006-01-02")).Scan(&count)
		if err != nil || count > 0 {
			http.Error(w, "Dokter tidak tersedia pada tanggal tersebut (libur).", http.StatusConflict) // 409 Conflict
			return
		}

		// 5. Pengecekan #2: Apakah sesuai dengan jadwal kerja mingguan?
		dayOfWeek := int(newDate.Weekday())
		if dayOfWeek == 0 {
			dayOfWeek = 7
		} // Konversi Minggu dari 0 ke 7

		var startTime, endTime time.Time
		err = dbpool.QueryRow(context.Background(), "SELECT start_time, end_time FROM doctor_schedules WHERE doctor_id = $1 AND day_of_week = $2", doctorID, dayOfWeek).Scan(&startTime, &endTime)

		requestTime := newDate.Format("15:04:05") // Format HH:MM:SS
		if err != nil || requestTime < startTime.Format("15:04:05") || requestTime > endTime.Format("15:04:05") {
			http.Error(w, "Jadwal yang diminta di luar jam kerja dokter.", http.StatusConflict)
			return
		}

		// 6. Pengecekan #3: Apakah bentrok dengan janji temu lain?
		err = dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM appointments WHERE doctor_id = $1 AND appointment_date = $2 AND id != $3", doctorID, newDate, appointmentID).Scan(&count)
		if err != nil || count > 0 {
			http.Error(w, "Slot waktu yang diminta sudah terisi. Silakan pilih jam lain.", http.StatusConflict)
			return
		}

		// --- AKHIR VALIDASI JADWAL ---

		// 7. Jika semua validasi lolos, update janji temu
		query := `UPDATE appointments SET appointment_date = $1, status = 'RESCHEDULED' 
                  WHERE id = $2 
                  RETURNING id, patient_id, doctor_id, appointment_date, status, created_at`

		var updatedAppt Appointment
		err = dbpool.QueryRow(context.Background(), query, newDate, appointmentID).Scan(&updatedAppt.ID, &updatedAppt.PatientID, &updatedAppt.DoctorID, &updatedAppt.AppointmentDate, &updatedAppt.Status, &updatedAppt.CreatedAt)
		if err != nil {
			log.Printf("Gagal update janji temu: %v", err)
			http.Error(w, "Gagal memperbarui janji temu", http.StatusInternalServerError)
			return
		}

		// 8. Kirim response sukses
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedAppt)
	}
}

// AddDoctorScheduleHandler menambahkan jadwal kerja mingguan untuk dokter.
func AddDoctorScheduleHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Ambil & Validasi ID Dokter dari URL
		doctorIDStr := r.PathValue("id")
		doctorID, err := strconv.Atoi(doctorIDStr)
		if err != nil {
			http.Error(w, "ID dokter tidak valid", http.StatusBadRequest)
			return
		}

		// 2. Dekode Request Body JSON
		var req ScheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Request body tidak valid", http.StatusBadRequest)
			return
		}

		// 3. Validasi Data dari Body
		// Validasi #1: Cek rentang hari
		if req.DayOfWeek < 1 || req.DayOfWeek > 7 {
			http.Error(w, "dayOfWeek harus antara 1 (Senin) dan 7 (Minggu).", http.StatusBadRequest)
			return
		}

		// Validasi #2 & #3: Cek format waktu
		timeLayout := "15:04:05" // Format HH:MM:SS
		startTime, err := time.Parse(timeLayout, req.StartTime)
		if err != nil {
			http.Error(w, "Format startTime tidak valid atau kosong, harus 'HH:MM:SS'", http.StatusBadRequest)
			return
		}
		endTime, err := time.Parse(timeLayout, req.EndTime)
		if err != nil {
			http.Error(w, "Format endTime tidak valid atau kosong, harus 'HH:MM:SS'", http.StatusBadRequest)
			return
		}

		// Validasi #4: Cek urutan waktu
		if startTime.After(endTime) || startTime.Equal(endTime) {
			http.Error(w, "startTime harus sebelum endTime.", http.StatusBadRequest)
			return
		}

		// 4. Masukkan Data ke Database
		query := `INSERT INTO doctor_schedules (doctor_id, day_of_week, start_time, end_time)
                  VALUES ($1, $2, $3, $4)`

		_, err = dbpool.Exec(context.Background(), query, doctorID, req.DayOfWeek, req.StartTime, req.EndTime)
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				http.Error(w, "Jadwal untuk hari ini sudah ada.", http.StatusConflict)
				return
			}
			log.Printf("Gagal menyimpan jadwal dokter: %v", err)
			http.Error(w, "Gagal menyimpan jadwal", http.StatusInternalServerError)
			return
		}

		// 5. Kirim Respons Sukses
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "Jadwal berhasil ditambahkan"}`))
	}
}

// GetDoctorSchedulesHandler mengambil jadwal kerja mingguan seorang dokter.
func GetDoctorSchedulesHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Ambil ID dokter dari URL
		doctorID := r.PathValue("id")

		// 2. Query untuk mengambil semua jadwal dokter tersebut
		query := `SELECT day_of_week, start_time, end_time FROM doctor_schedules WHERE doctor_id = $1`

		rows, err := dbpool.Query(context.Background(), query, doctorID)
		if err != nil {
			http.Error(w, "Gagal mengambil data jadwal", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// 3. Looping melalui hasil dan masukkan ke dalam slice
		var schedules []ScheduleResponse
		for rows.Next() {
			var s ScheduleResponse
			var startTime, endTime time.Time // Tampung sebagai time.Time dulu
			if err := rows.Scan(&s.DayOfWeek, &startTime, &endTime); err != nil {
				http.Error(w, "Gagal memindai data jadwal", http.StatusInternalServerError)
				return
			}
			// Format ke string HH:MM:SS
			s.StartTime = startTime.Format("15:04:05")
			s.EndTime = endTime.Format("15:04:05")
			schedules = append(schedules, s)
		}

		if schedules == nil {
			schedules = []ScheduleResponse{}
		}

		// 4. Kirim response JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(schedules)
	}
}

// AddDoctorTimeOffHandler menambahkan tanggal libur untuk dokter.
func AddDoctorTimeOffHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doctorID := r.PathValue("id")

		var req TimeOffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Request body tidak valid", http.StatusBadRequest)
			return
		}

		// Validasi format tanggal
		layout := "2006-01-02" // Format YYYY-MM-DD
		offDate, err := time.Parse(layout, req.OffDate)
		if err != nil {
			http.Error(w, "Format tanggal harus YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		// Masukkan data ke database
		query := `INSERT INTO doctor_time_off (doctor_id, off_date, reason) VALUES ($1, $2, $3)`

		_, err = dbpool.Exec(context.Background(), query, doctorID, offDate, req.Reason)
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				http.Error(w, "Tanggal libur ini sudah terdaftar.", http.StatusConflict)
				return
			}
			log.Printf("Gagal menyimpan tanggal libur: %v", err)
			http.Error(w, "Gagal menyimpan tanggal libur", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "Tanggal libur berhasil ditambahkan"}`))
	}
}
