#!/bin/bash

# Pastikan script dijalankan sebagai root
if [ "$EUID" -ne 0 ]; then
  echo "❌ Harap jalankan sebagai root (sudo su)"
  exit
fi

echo "🚀 Memulai Instalasi GoPanel untuk AlmaLinux..."

# 1. SETUP SELINUX (PENTING DI ALMALINUX)
# Kita set ke Permissive dulu biar Nginx bisa baca file yang dibuat GoPanel
echo "🛡️ Mengatur SELinux..."
setenforce 0
sed -i 's/^SELINUX=.*/SELINUX=permissive/g' /etc/selinux/config

# 2. INSTALL DEPENDENCIES
echo "📦 Menginstall Software (Nginx, MariaDB, PHP, Go)..."
dnf install epel-release -y
dnf install nginx mariadb-server git golang -y
# Install PHP 8.x (AlmaLinux 9 defaultnya php 8.0/8.1)
dnf install php php-fpm php-mysqlnd php-json php-mbstring -y

# 3. SETUP NGINX CONFIGURATION (OVERWRITE METHOD)
# AlmaLinux gapunya sites-available, kita bikin manual biar cocok sama GoPanel
echo "🔧 Mengatur Konfigurasi Nginx..."
mkdir -p /etc/nginx/sites-available
mkdir -p /etc/nginx/sites-enabled
mkdir -p /var/www

# Backup config asli bawaan AlmaLinux
mv /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak.$(date +%F)

# Buat Config Nginx Baru yang BERSIH & STANDARD
cat <<EOF > /etc/nginx/nginx.conf
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log notice;
pid /run/nginx.pid;

# Load dynamic modules
include /usr/share/nginx/modules/*.conf;

events {
    worker_connections 1024;
}

http {
    log_format  main  '\$remote_addr - \$remote_user [\$time_local] "\$request" '
                      '\$status \$body_bytes_sent "\$http_referer" '
                      '"\$http_user_agent" "\$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile            on;
    tcp_nopush          on;
    tcp_nodelay         on;
    keepalive_timeout   65;
    types_hash_max_size 2048;
    client_max_body_size 100M; # Biar bisa upload file agak gede
    server_names_hash_bucket_size 64; # Biar gak error kalau domain panjang

    include             /etc/nginx/mime.types;
    default_type        application/octet-stream;

    # Load modular configuration files from the /etc/nginx/conf.d directory.
    # See http://nginx.org/en/docs/ngx_core_module.html#include
    # for more information.
    include /etc/nginx/conf.d/*.conf;

    # === GOPANEL CONFIG ===
    # Ini baris paling penting! Baca config website user di sini.
    include /etc/nginx/sites-enabled/*.conf;

    # KITA TIDAK MEMASUKKAN BLOK SERVER DEFAULT DI SINI
    # JADI WEBSITE PERTAMA YANG DIBUAT USER AKAN LANGSUNG JALAN
}
EOF

# Hapus default.conf di folder conf.d jika ada (kadang ini juga jadi pengganggu)
rm -f /etc/nginx/conf.d/default.conf

# Edit nginx.conf biar baca folder sites-enabled
# Kita cari baris 'include /etc/nginx/conf.d/*.conf;' dan tambahkan sites-enabled di bawahnya
if ! grep -q "sites-enabled" /etc/nginx/nginx.conf; then
    sed -i '/conf.d\/\*\.conf;/a \    include /etc/nginx/sites-enabled/*.conf;' /etc/nginx/nginx.conf
fi

# 4. SETUP PHP-FPM
# Ubah user PHP-FPM jadi nginx (defaultnya apache di RHEL)
echo "🐘 Mengatur PHP-FPM..."
sed -i 's/user = apache/user = nginx/g' /etc/php-fpm.d/www.conf
sed -i 's/group = apache/group = nginx/g' /etc/php-fpm.d/www.conf
# Pastikan PHP-FPM listen di socket (bukan TCP) agar sesuai template GoPanel kamu
# Kita paksa pakai socket unix:/run/php-fpm/www.sock
sed -i 's/^listen =.*/listen = \/run\/php-fpm\/www.sock/g' /etc/php-fpm.d/www.conf
# Set permission socket
sed -i 's/;listen.owner = nobody/listen.owner = nginx/g' /etc/php-fpm.d/www.conf
sed -i 's/;listen.group = nobody/listen.group = nginx/g' /etc/php-fpm.d/www.conf
sed -i 's/;listen.mode = 0660/listen.mode = 0660/g' /etc/php-fpm.d/www.conf

# Karena di template Go kamu path socketnya beda (unix:/run/php/php8.x...),
# Kita buat symlink biar GoPanel gak bingung.
mkdir -p /run/php
ln -sf /run/php-fpm/www.sock /run/php/php8.1-fpm.sock
# (Catatan: Kamu mungkin perlu update template Go nanti untuk menyesuaikan ini)

# 5. START SERVICES
echo "🔥 Menyalakan Service..."
systemctl enable --now nginx
systemctl enable --now mariadb
systemctl enable --now php-fpm

# 6. FIREWALL (Firewalld)
echo "🧱 Membuka Port Firewall..."
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --permanent --add-port=3000/tcp # Port Panel
firewall-cmd --reload

# 7. BUILD GOPANEL
echo "🔍 Mendeteksi versi Go di sistem..."
# 1. Ambil versi Go terinstall (Output: go1.25.3 -> 1.25.3)
INSTALLED_VER=$(go version | awk '{print $3}' | sed 's/go//')

# 2. Ambil cuma angka Major.Minor (1.25.3 -> 1.25)
# Karena go.mod butuhnya format 1.xx
SAFE_VER=$(echo $INSTALLED_VER | cut -d. -f1,2)

echo "✨ Versi terdeteksi: $INSTALLED_VER. Menyesuaikan go.mod ke versi $SAFE_VER..."
# Hapus go.mod lama biar fresh (opsional, jaga2 conflict versi Go)
rm -f go.sum
# Perintah sakti untuk ubah versi di go.mod
go mod edit -go=$SAFE_VER
go mod tidy
go build -o gopanel-server main.go

# Cek apakah file berhasil dibuat
if [ ! -f "gopanel-server" ]; then
    echo "❌ Gagal Compile! Cek error di atas."
    exit 1
fi

# 8. BUAT SYSTEMD SERVICE (Biar auto-start pas restart VPS)
echo "⚙️ Membuat Service Systemd..."
cat <<EOF > /etc/systemd/system/gopanel.service
[Unit]
Description=GoPanel Hosting Control Panel
After=network.target mysql.service nginx.service

[Service]
User=root
Group=root
WorkingDirectory=$(pwd)
ExecStart=$(pwd)/gopanel-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Reload Daemon & Start
systemctl daemon-reload
systemctl enable --now gopanel

echo "=================================================="
echo "✅ INSTALASI SELESAI!"
echo "--------------------------------------------------"
echo "Panel akses di: http://$(curl -s ifconfig.me):3000"
echo "Login default : admin / admin123"
echo "=================================================="