#!/bin/bash

echo "🚀 MEMULAI UPDATE GOPANEL..."

# 1. AMBIL KODINGAN TERBARU
echo "📥 Pulling from Git..."
git pull origin main
# Catatan: Kalau branch kamu bukan 'main' (misal 'master'), ganti jadi 'git pull origin master'

# 2. FIX VERSI GO (PENTING! Karena git pull akan menimpa go.mod)
# Kita ulangi trik deteksi versi biar compile gak error
echo "🔧 Menyesuaikan versi Go..."
INSTALLED_VER=$(go version | awk '{print $3}' | sed 's/go//')
SAFE_VER=$(echo $INSTALLED_VER | cut -d. -f1,2)
go mod edit -go=$SAFE_VER
go mod tidy

# 3. COMPILE ULANG (MASAK ULANG)
echo "🔨 Building Binary..."
go build -o gopanel-server main.go

# Cek kalau build gagal, jangan restart service (biar website gak mati)
if [ $? -ne 0 ]; then
    echo "❌ BUILD GAGAL! Service tidak direstart."
    exit 1
fi

# 4. RESTART SERVICE
echo "🔄 Restarting Service..."
systemctl restart gopanel

# 5. CEK STATUS
if systemctl is-active --quiet gopanel; then
    echo "✅ UPDATE SUKSES! Server sudah jalan dengan codingan baru."
else
    echo "⚠️ Service mati. Cek 'systemctl status gopanel' untuk detail."
fi