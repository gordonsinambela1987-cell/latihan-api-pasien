package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Patient merepresentasikan struktur data untuk seorang pasien.
// Tanda `json:"..."` digunakan untuk mengontrol bagaimana data ini
// diubah dari dan ke format JSON saat berkomunikasi via API.
type Patient struct {
	ID          int       `json:"id"`
	KTPNumber   string    `json:"ktpNumber"`
	FullName    string    `json:"fullName"`
	DateOfBirth time.Time `json:"dateOfBirth"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CreatePatientHandler adalah fungsi yang menangani pembuatan pasien baru.
// Perhatikan bagaimana kita memberikan `dbpool` agar handler ini bisa berbicara dengan database.
func CreatePatientHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Dekode request JSON yang masuk ke dalam struct Patient
		var p Patient
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Request body tidak valid", http.StatusBadRequest)
			return
		}

		// 2. Validasi input (Praktik Kualitas Kode yang Baik)
		if len(p.KTPNumber) != 16 {
			http.Error(w, "Nomor KTP harus 16 digit", http.StatusBadRequest)
			return
		}
		if len(p.FullName) < 3 {
			http.Error(w, "Nama lengkap minimal 3 karakter", http.StatusBadRequest)
			return
		}

		// 3. Masukkan data ke database
		// Kita gunakan QueryRow untuk langsung mendapatkan ID pasien yang baru dibuat.
		query := `INSERT INTO patients (ktp_number, full_name, date_of_birth) 
                  VALUES ($1, $2, $3) 
                  RETURNING id, created_at`

		err := dbpool.QueryRow(context.Background(), query, p.KTPNumber, p.FullName, p.DateOfBirth).Scan(&p.ID, &p.CreatedAt)
		if err != nil {
			// (Nanti kita bisa tambahkan pengecekan error duplikat KTP di sini)
			log.Printf("Gagal memasukkan pasien ke DB: %v", err)
			http.Error(w, "Gagal menyimpan data pasien", http.StatusInternalServerError)
			return
		}

		// 4. Kirim response JSON yang sukses
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // Status 201 Created
		json.NewEncoder(w).Encode(p)
	}
}

// GetPatientByIDHandler adalah fungsi untuk mengambil satu pasien berdasarkan ID.
func GetPatientByIDHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Ambil ID dari URL path
		id := r.PathValue("id")

		// 2. Query ke database untuk mendapatkan pasien
		var p Patient
		query := `SELECT id, ktp_number, full_name, date_of_birth, created_at 
                  FROM patients 
                  WHERE id = $1`

		err := dbpool.QueryRow(context.Background(), query, id).Scan(&p.ID, &p.KTPNumber, &p.FullName, &p.DateOfBirth, &p.CreatedAt)
		if err != nil {
			// Jika error-nya adalah "no rows", berarti pasien tidak ditemukan
			if err.Error() == "no rows in result set" {
				http.Error(w, "Pasien tidak ditemukan", http.StatusNotFound) // Status 404
				return
			}
			fmt.Println(err)
			// Untuk error database lainnya
			http.Error(w, "Gagal mengambil data pasien", http.StatusInternalServerError)
			return
		}

		// 3. Kirim response JSON yang sukses
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	}
}
