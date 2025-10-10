package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

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

		// Masukkan data ke database menggunakan tanggal yang sudah dikonversi
		query := `INSERT INTO patients (ktp_number, full_name, date_of_birth) 
                  VALUES ($1, $2, $3) 
                  RETURNING id, created_at`

		err = dbpool.QueryRow(context.Background(), query, p.KTPNumber, p.FullName, dob).Scan(&p.ID, &p.CreatedAt)
		if err != nil {
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
		if len(d.Name) < 3 {
			http.Error(w, "Nama dokter minimal 3 karakter", http.StatusBadRequest)
			return
		}

		// 3. Masukkan data ke database
		query := `INSERT INTO doctors (nik, name, specialty) 
                  VALUES ($1, $2, $3) 
                  RETURNING id`

		err := dbpool.QueryRow(context.Background(), query, d.NIK, d.Name, d.Specialty).Scan(&d.ID)
		if err != nil {
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
