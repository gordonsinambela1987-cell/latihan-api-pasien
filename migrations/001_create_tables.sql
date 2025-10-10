-- Membuat Tabel Pasien
CREATE TABLE patients (
    id SERIAL PRIMARY KEY,
    ktp_number VARCHAR(16) NOT NULL UNIQUE,
    full_name VARCHAR(100) NOT NULL,
    date_of_birth DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Membuat Tabel Dokter
CREATE TABLE doctors (
    id SERIAL PRIMARY KEY,
    nik VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    specialty VARCHAR(100) NOT NULL
);

-- Membuat Tabel Janji Temu
CREATE TABLE appointments (
    id SERIAL PRIMARY KEY,
    patient_id INTEGER NOT NULL REFERENCES patients(id),
    doctor_id INTEGER NOT NULL REFERENCES doctors(id),
    appointment_date TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'CONFIRMED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Membuat Tabel Jadwal Dokter
CREATE TABLE doctor_schedules (
    id SERIAL PRIMARY KEY,
    doctor_id INTEGER NOT NULL REFERENCES doctors(id),
    day_of_week INTEGER NOT NULL, -- 1 for Monday, 7 for Sunday
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    UNIQUE (doctor_id, day_of_week)
);

-- Membuat Tabel Hari Libur Dokter
CREATE TABLE doctor_time_off (
    id SERIAL PRIMARY KEY,
    doctor_id INTEGER NOT NULL REFERENCES doctors(id),
    off_date DATE NOT NULL,
    reason VARCHAR(255),
    UNIQUE (doctor_id, off_date)
);