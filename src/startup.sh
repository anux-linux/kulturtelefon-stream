#!/bin/sh

exec ./kulturtelefon --config ${CONFIG_PATH:-/app/config/stream.config} --loglevel ${LOG_LEVEL:-info}