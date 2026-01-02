server {
    listen 80;
    server_name {{ .Domain }};

    # Folder Root Project
    root /var/www/{{ .Domain }}/public_html;
    index index.php index.html index.htm;

    access_log /var/log/nginx/{{ .Domain }}.access.log;
    error_log /var/log/nginx/{{ .Domain }}.error.log;

    {{ if eq .Type "proxy" }}
    # === KONFIGURASI UNTUK NODE.JS / PYTHON / GO ===
    location / {
        proxy_pass http://127.0.0.1:{{ .Port }};
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
    {{ else }}
    # === KONFIGURASI UNTUK PHP & HTML STATIC ===
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    # Blok khusus PHP (Hanya aktif jika user memilih PHP)
    {{ if eq .Type "php" }}
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        # Arahkan ke socket PHP (Contoh PHP 8.1, nanti bisa dibikin dinamis juga)
        fastcgi_pass unix:/run/php/php8.1-fpm.sock;
    }
    {{ end }}
    {{ end }}
}