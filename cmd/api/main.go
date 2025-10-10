package main

import (
	"log"
	"net/http"

	"github.com/gordonsinambela1987-cell/latihan-api-pasien-go/internal/database"
	// Import package handlers kita
	"github.com/gordonsinambela1987-cell/latihan-api-pasien-go/internal/handlers"
)

func main() {
	dbPool := database.Connect()
	defer dbPool.Close()

	router := http.NewServeMux()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Selamat Datang di API Pasien v1"))
	})

	// DAFTARKAN ENDPOINT BARU DI SINI
	// Enpoin pendaftaran data doctor
	router.HandleFunc("POST /doctors", handlers.CreateDoctorHandler(dbPool))
	// Enpoin pendaftaran pasien
	router.HandleFunc("POST /patients", handlers.CreatePatientHandler(dbPool))
	// Endpoint untuk mengambil data satu pasien berdasarkan ID
	router.HandleFunc("GET /patients/{id}", handlers.GetPatientByIDHandler(dbPool))
	// Endpoint untuk mengambil data doctor
	router.HandleFunc("GET /doctors", handlers.GetAllDoctorsHandler(dbPool))

	port := ":8080"
	server := &http.Server{
		Addr:    port,
		Handler: router,
	}

	log.Printf("Server dimulai di port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Gagal memulai server: %s\n", err)
	}
}
