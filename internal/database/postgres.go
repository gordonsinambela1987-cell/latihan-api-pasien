package database

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Fungsi Connect akan membuat dan mengembalikan "connection pool".
// Connection pool jauh lebih efisien daripada membuat koneksi baru untuk setiap request.
func Connect() *pgxpool.Pool {
	// Untuk saat ini, kita tulis langsung URL koneksi database-nya.
	// Nanti kita akan belajar cara memuat ini dari file .env agar lebih aman.
	dbURL := "postgres://postgres:mysecretpassword@localhost:5432/postgres"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Tidak dapat membuat connection pool: %v\n", err)
		os.Exit(1)
	}

	// Lakukan ping untuk memastikan koneksi ke database berhasil.
	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatalf("Tidak dapat terhubung ke database: %v\n", err)
		os.Exit(1)
	}

	log.Println("Berhasil terhubung ke database!")
	return pool
}