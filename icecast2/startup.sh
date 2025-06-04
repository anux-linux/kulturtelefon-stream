#!/bin/sh

env

set -x

set_value() {
    if [ -n "$2" ]; then
        echo "set '$2' to '$1'"
        sed -i "s/<$2>[^<]*<\/$2>/<$2>$1<\/$2>/g" /etc/icecast2/icecast.xml
    else
        echo "Setting for '$1' is missing, skipping." >&2
    fi
}

# Set default values if environment variables are not provided
ICECAST_SOURCE_PASSWORD=${ICECAST_SOURCE_PASSWORD:-hackme}
ICECAST_RELAY_PASSWORD=${ICECAST_RELAY_PASSWORD:-hackme}
ICECAST_ADMIN_PASSWORD=${ICECAST_ADMIN_PASSWORD:-hackme}
ICECAST_PASSWORD=${ICECAST_PASSWORD:-hackme}
ICECAST_HOSTNAME=${ICECAST_HOSTNAME:-icecast2}

set_value $ICECAST_SOURCE_PASSWORD source-password
set_value $ICECAST_RELAY_PASSWORD  relay-password
set_value $ICECAST_ADMIN_PASSWORD  admin-password
set_value $ICECAST_PASSWORD        password
set_value $ICECAST_HOSTNAME        hostname


set -e

chown -R icecast2 /var/log/icecast2
chown -R icecast2 /etc/icecast2

icecast2 icecast2 -n -c /etc/icecast2/icecast.xml