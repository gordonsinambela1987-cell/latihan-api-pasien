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

	// --- Endpoints Pasien ---
	router.HandleFunc("POST /patients", handlers.CreatePatientHandler(dbPool))
	router.HandleFunc("GET /patients/{id}", handlers.GetPatientByIDHandler(dbPool))

	// --- Endpoints Dokter ---
	router.HandleFunc("GET /doctors", handlers.GetAllDoctorsHandler(dbPool))
	router.HandleFunc("POST /doctors", handlers.CreateDoctorHandler(dbPool))
	// --- Endpoints Jadwal Kerja Dokter ---
	router.HandleFunc("POST /doctors/{id}/schedules", handlers.AddDoctorScheduleHandler(dbPool))
	router.HandleFunc("GET /doctors/{id}/schedules", handlers.GetDoctorSchedulesHandler(dbPool))
	router.HandleFunc("POST /doctors/{id}/timeoff", handlers.AddDoctorTimeOffHandler(dbPool))

	// --- Endpoint Janji Temu ---
	router.HandleFunc("POST /appointments", handlers.CreateAppointmentHandler(dbPool))
	router.HandleFunc("GET /patients/{id}/appointments", handlers.GetAppointmentsByPatientIDHandler(dbPool))
	router.HandleFunc("PATCH /appointments/{id}", handlers.RescheduleAppointmentHandler(dbPool))

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
