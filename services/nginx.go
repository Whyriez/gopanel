package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

type NginxData struct {
	Domain string
	Type   string
	Port   string
}

func GenerateNginxConfig(domain string, typeSite string, port string) (string, error) {
	// === 1. TENTUKAN PATH ASLI LINUX ===
	// Lokasi config Nginx di VPS (Standard Ubuntu/Debian)
	availablePath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", domain)
	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", domain)

	// Lokasi Folder Website (Root Directory)
	webRoot := fmt.Sprintf("/var/www/%s/public_html", domain)

	// === 2. BIKIN FOLDER WEBSITE ===
	// Sama kayak mkdir -p /var/www/domain.com/public_html
	// Permission 0755 biar bisa dibaca Nginx
	if err := os.MkdirAll(webRoot, 0755); err != nil {
		return "", fmt.Errorf("gagal buat folder webroot: %v", err)
	}

	// Buat file index.html dummy biar pas dibuka gak 40
	dummyIndex := []byte("<h1>Halo! Website " + domain + " berhasil dibuat via GoPanel 🚀</h1>")
	os.WriteFile(filepath.Join(webRoot, "index.html"), dummyIndex, 0644)

	// === 3. UPDATE PERMISSION (CHOWN) ===
	// Folder web harus milik 'www-data' (user standard Nginx)
	// Kita jalankan command linux: chown -R www-data:www-data /var/www/domain.com
	cmdChown := exec.Command("chown", "-R", "www-data:www-data", fmt.Sprintf("/var/www/%s", domain))
	cmdChown.Run() // Kita ignore error dulu kalau user www-data gak ketemu (opsional)

	// === 4. GENERATE CONFIG NGINX ===
	// Baca template
	cwd, _ := os.Getwd()
	tplPath := filepath.Join(cwd, "templates", "nginx_site.tpl")

	tpl, err := template.ParseFiles(tplPath)
	if err != nil {
		return "", fmt.Errorf("gagal baca template: %v", err)
	}

	// Tulis langsung ke /etc/nginx/sites-available/domain.com
	file, err := os.Create(availablePath)
	if err != nil {
		return "", fmt.Errorf("gagal tulis ke /etc/nginx (Apakah jalan sebagai root?): %v", err)
	}

	data := NginxData{Domain: domain, Type: typeSite, Port: port}
	err = tpl.Execute(file, data)
	file.Close() // Close manual biar bisa disave sebelum direload
	if err != nil {
		return "", fmt.Errorf("gagal isi config: %v", err)
	}

	// === 5. AKTIFKAN CONFIG (SYMLINK) ===
	// Command: ln -s /etc/nginx/sites-available/domain /etc/nginx/sites-enabled/domain
	// Hapus dulu kalau symlink lama ada (biar gak error file exist)
	os.Remove(enabledPath)
	err = os.Symlink(availablePath, enabledPath)
	if err != nil {
		return "", fmt.Errorf("gagal symlink: %v", err)
	}

	// === 6. RELOAD NGINX ===
	// Command: systemctl reload nginx
	cmd := exec.Command("systemctl", "reload", "nginx")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gagal reload nginx (Cek config error?): %v", err)
	}

	return availablePath, nil
}
