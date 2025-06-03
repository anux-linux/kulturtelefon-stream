#!/bin/bash

APP_DIR=$1

if [ -z "$APP_DIR" ]; then
    echo "Usage: $0 <path_to_app_directory>"
    exit 1
fi

# Check if CERT_DOMAINS and CERT_EMAILS are set
if [ -z "$CERT_DOMAINS" ]; then
    echo "CERT_DOMAINS is not set. Please set it and try again."
    exit 1
fi

if [ -z "$CERT_EMAIL" ]; then
    echo "CERT_EMAIL is not set. Please set it and try again."
    exit 1
fi

# Create directories for certbot if they don't exist
mkdir -p $APP_DIR/docker/nginx/conf.d
mkdir -p $APP_DIR/docker/certbot/certs/live/$CERT_DOMAINS

# Create dummy certificates for initial nginx startup
if [ ! -f "$APP_DIR/docker/certbot/certs/live/$CERT_DOMAINS/fullchain.pem" ]; then
    echo "Creating dummy certificate for $CERT_DOMAINS..."
    mkdir -p $APP_DIR/docker/certbot/certs/live/$CERT_DOMAINS
    
    openssl req -x509 -nodes -newkey rsa:2048 \
        -days 1 \
        -keyout $APP_DIR/docker/certbot/certs/live/$CERT_DOMAINS/privkey.pem \
        -out $APP_DIR/docker/certbot/certs/live/$CERT_DOMAINS/fullchain.pem \
        -subj "/CN=$CERT_DOMAINS"
fi

# Create the initial nginx configuration for HTTP challenge ONLY
cat > $APP_DIR/docker/nginx/nginx.conf << 'EOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    
    access_log /var/log/nginx/access.log main;
    
    sendfile on;
    keepalive_timeout 65;
    
    # HTTP server for ACME challenge
    server {
        listen 80;
        server_name DOMAIN_PLACEHOLDER;
        
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }
        
        location / {
            return 200 'Nginx is running. SSL setup in progress...';
            add_header Content-Type text/plain;
        }
    }
}
EOF

# Replace domain placeholders
sed -i "s/DOMAIN_PLACEHOLDER/$CERT_DOMAINS/g" $APP_DIR/docker/nginx/nginx.conf

echo "Starting nginx with HTTP-only config..."
# Start nginx to handle the ACME challenge
docker-compose up -d nginx

# Wait for nginx to be ready
sleep 5

echo "Requesting SSL certificate for $CERT_DOMAINS..."
# Remove any existing certificates first to force a fresh request
rm -rf $APP_DIR/docker/certbot/certs/live/$CERT_DOMAINS
rm -rf $APP_DIR/docker/certbot/certs/archive/$CERT_DOMAINS
rm -f $APP_DIR/docker/certbot/certs/renewal/$CERT_DOMAINS.conf

# Request the certificate (without --force-renewal since we're starting fresh)
docker-compose run --rm --entrypoint "\
  certbot certonly --webroot -w /var/www/certbot \
    $staging_arg \
    $email_arg \
    $domain_args \
    --rsa-key-size $rsa_key_size \
    --agree-tos \
    --force-renewal" certbot

echo "Certificate request completed. Checking for errors..."

# Check if certificate was created successfully
if [ ! -f "$APP_DIR/docker/certbot/certs/live/$CERT_DOMAINS/fullchain.pem" ]; then
    echo "Certificate generation failed. Check the logs above."
    exit 1
fi

echo "Certificate obtained successfully. Creating full nginx config with SSL..."

# Create the final nginx config with SSL
cat > $APP_DIR/docker/nginx/nginx.conf << 'EOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    
    access_log /var/log/nginx/access.log main;
    
    sendfile on;
    keepalive_timeout 65;
    
    # HTTPS parameters
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers "EECDH+AESGCM:EDH+AESGCM:AES256+EECDH:AES256+EDH";
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    ssl_session_tickets off;
    ssl_stapling on;
    ssl_stapling_verify on;
    
    # Compression
    gzip on;
    gzip_disable "msie6";
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    
    # HTTP -> HTTPS redirect and ACME challenge
    server {
        listen 80;
        server_name DOMAIN_PLACEHOLDER;
        
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }
        
        location / {
            return 301 https://$host$request_uri;
        }
    }
    
    # HTTPS server
    server {
        listen 443 ssl;
        http2 on;
        server_name DOMAIN_PLACEHOLDER;
        
        # SSL certificates
        ssl_certificate /etc/nginx/ssl/live/DOMAIN_PLACEHOLDER/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/live/DOMAIN_PLACEHOLDER/privkey.pem;
        
        # Proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Stream API
        location /api/ {
            proxy_pass http://stream-api:8080/;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
        
        # Icecast streaming - audio content
        location /stream/ {
            proxy_pass http://icecast:8000/;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            
            # Important for streaming
            proxy_buffering off;
            proxy_cache off;
            proxy_read_timeout 36000s;
        }
        
        # Admin interface for Icecast
        location /admin/ {
            proxy_pass http://icecast:8000/admin/;
            proxy_http_version 1.1;
        }
        
        # Status and stats pages for Icecast
        location /status {
            proxy_pass http://icecast:8000/status;
            proxy_http_version 1.1;
        }
        
        location /status-json.xsl {
            proxy_pass http://icecast:8000/status-json.xsl;
            proxy_http_version 1.1;
        }
        
        # Root path with directory listing
        location / {
            return 200 '<!DOCTYPE html>
                    <html lang="de">
                    <head>
                    <meta charset="UTF-8">
                    <meta name="viewport" content="width=device-width, initial-scale=1.0">
                    <title>Kulturtelefon Streaming Service</title>
                    <style>
                        body {
                        font-family: "Helvetica Neue", Arial, sans-serif;
                        margin: 0;
                        padding: 0;
                        min-height: 100vh;
                        display: flex;
                        justify-content: center;
                        align-items: center;
                        background-color: #f5f5f5;
                        color: #333;
                        }
                        
                        .container {
                        text-align: center;
                        background-color: white;
                        padding: 2rem 3rem;
                        border-radius: 8px;
                        box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
                        max-width: 90%;
                        width: 500px;
                        }
                        
                        h1 {
                        margin-top: 0;
                        color: #2c3e50;
                        font-size: 2rem;
                        }
                        
                        .links {
                        list-style: none;
                        padding: 0;
                        margin: 2rem 0;
                        display: flex;
                        flex-direction: column;
                        gap: 0.8rem;
                        }
                        
                        .links a {
                        display: inline-block;
                        text-decoration: none;
                        color: #3498db;
                        font-weight: 500;
                        padding: 0.5rem 1rem;
                        border-radius: 4px;
                        transition: all 0.3s ease;
                        }
                        
                        .links a:hover {
                        background-color: #f0f7ff;
                        color: #2980b9;
                        }
                        
                        .links a.primary {
                        background-color: #3498db;
                        color: white;
                        padding: 0.7rem 1.5rem;
                        }
                        
                        .links a.primary:hover {
                        background-color: #2980b9;
                        }
                        
                        .logo {
                        margin-bottom: 1.5rem;
                        font-size: 2.5rem;
                        color: #2c3e50;
                        }
                        
                        footer {
                        margin-top: 2rem;
                        font-size: 0.8rem;
                        color: #7f8c8d;
                        }
                    </style>
                    </head>
                    <body>
                    <div class="container">
                        <div class="logo">&#128266;</div>
                        <h1>Kulturtelefon Streaming Service</h1>
                        <ul class="links">
                        <li><a href="https://www.kulturtelefon.de">Weitere Informationen</a></li>
                        </ul>
                        <footer>
                        © 2025 Kulturtelefon Streaming Service
                        </footer>
                    </div>
                    </body>
                </html>';
            default_type text/html;
        }
        
        # Error pages
        error_page 500 502 503 504 /50x.html;
        location = /50x.html {
            root /usr/share/nginx/html;
        }
    }
}
EOF

# Replace domain placeholders
sed -i "s/DOMAIN_PLACEHOLDER/$CERT_DOMAINS/g" $APP_DIR/docker/nginx/nginx.conf

echo "Reloading nginx with SSL configuration..."
# Reload nginx configuration
docker-compose exec nginx nginx -s reload

echo "Letsencrypt script completed successfully!"
echo "Your site should now be available at https://$CERT_DOMAINS"