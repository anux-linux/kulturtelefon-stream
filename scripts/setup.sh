#!/bin/bash

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker and try again."
    exit 1
fi

APP_DIR="/etc/stream-api"

echo "Setting up directories..."
# Creating docker directories
mkdir -p $APP_DIR/docker/config
mkdir -p $APP_DIR/docker/logs/stream-api
mkdir -p $APP_DIR/docker/data/stream-api

mkdir -p $APP_DIR/docker/icecast/mounts
mkdir -p $APP_DIR/docker/logs/icecast
mkdir -p $APP_DIR/docker/data/icecast

mkdir -p $APP_DIR/docker/nginx/conf.d
mkdir -p $APP_DIR/docker/certbot/www
mkdir -p $APP_DIR/docker/certbot/certs

echo "Directories set up successfully."

echo "Coping default templates..."
cp -r ./templates $APP_DIR/docker/data/templates

echo "Templates copied successfully."


echo "Setting up stream-api confg..."

# Generate a random 32-byte key and convert to hex
RANDOM_KEY=$(openssl rand -hex 32)
if [ $? -ne 0 ]; then
    echo "Failed to generate random key"
    exit 1
fi

ADMIN_USER="admin"
ADMIN_PASS=$(openssl rand -base64 12)
if [ $? -ne 0 ]; then
    echo "Failed to generate random password"
    exit 1
fi

cat > $APP_DIR/docker/config/stream.config << 'EOF'
icecast_mounts_folder: /app/icecast/mounts
db_file: /app/data/streams.db
default_mount_template: /app/data/templates/default_mount.tmpl
private_mount_template: /app/data/templates/private_mount.tmpl
secret_key: RANDOM_KEY_PLACEHOLDER
admin_username: ADMIN_USER_PLACEHOLDER
admin_password: ADMIN_PASS_PLACEHOLDER
EOF

sed -i "s/RANDOM_KEY_PLACEHOLDER/$RANDOM_KEY/g" $APP_DIR/docker/config/stream.config
sed -i "s/ADMIN_USER_PLACEHOLDER/$ADMIN_USER/g" $APP_DIR/docker/config/stream.config
sed -i "s/ADMIN_PASS_PLACEHOLDER/$ADMIN_PASS/g" $APP_DIR/docker/config/stream.config

echo "Stream-api config set up successfully."

echo "Init letsencrypt..."


./init-letsencryp.sh $APP_DIR
if [ $? -ne 0 ]; then
    echo "Failed to initialize Let's Encrypt"
    exit 1
fi
echo "Let's Encrypt initialized successfully."



echo "Starting docker containers..."

docker-compose up -d
if [ $? -ne 0 ]; then
    echo "Failed to start docker containers"
    exit 1
fi
echo "Docker containers started successfully."
echo "Waiting for stream-api to start..."
# Wait for stream-api to start  
while ! docker-compose exec stream-api curl -s http://localhost:8080/health | grep -q "UP"; do
    sleep 5
done
echo "Stream-api is up and running."

