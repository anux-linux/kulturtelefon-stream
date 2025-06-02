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

set_value $ICECAST_SOURCE_PASSWORD source-password
set_value $ICECAST_RELAY_PASSWORD  relay-password
set_value $ICECAST_ADMIN_PASSWORD  admin-password
set_value $ICECAST_PASSWORD        password
set_value $ICECAST_HOSTNAME        hostname

set -e

icecast2 icecast2 -n -c /etc/icecast2/icecast.xml