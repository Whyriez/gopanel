package services

import (
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/go-sql-driver/mysql" // Driver MySQL
)

// GANTI SESUAI PASSWORD ROOT MYSQL KOMPUTER KAMU!
const (
	dbUser = "root"
	dbPass = "" // Isi password root mysql kamu di sini (misal: "root" atau "123456")
	dbHost = "tcp(127.0.0.1:3306)"
)

// Validasi Nama (Hanya boleh huruf dan angka, Anti SQL Injection!)
func isValidName(name string) bool {
	match, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", name)
	return match
}

func CreateMySQLDatabase(dbName, dbUserNew, dbPassNew string) error {
	// 1. Security Check (Wajib!)
	if !isValidName(dbName) || !isValidName(dbUserNew) {
		return fmt.Errorf("nama database/user hanya boleh huruf, angka, dan underscore")
	}

	// 2. Konek ke MySQL sebagai ROOT
	dsn := fmt.Sprintf("%s:%s@%s/", dbUser, dbPass, dbHost)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// 3. Eksekusi Perintah (DDL)
	// Hati-hati: DDL tidak support prepared statement di beberapa driver,
	// makanya kita validasi regex di awal dengan ketat.

	// A. Buat Database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE `%s`", dbName))
	if err != nil {
		return fmt.Errorf("gagal create DB: %v", err)
	}

	// B. Buat User Database
	// Syntax MySQL 8.0+ dan MariaDB agak beda dikit, ini versi umum (MariaDB/MySQL 5.7+)
	_, err = db.Exec(fmt.Sprintf("CREATE USER '%s'@'localhost' IDENTIFIED BY '%s'", dbUserNew, dbPassNew))
	if err != nil {
		return fmt.Errorf("gagal create User: %v", err)
	}

	// C. Kasih Izin (Grant All)
	_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'", dbName, dbUserNew))
	if err != nil {
		return fmt.Errorf("gagal grant privileges: %v", err)
	}

	// D. Flush
	db.Exec("FLUSH PRIVILEGES")

	return nil
}
