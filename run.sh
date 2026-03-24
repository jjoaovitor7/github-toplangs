#!/usr/bin/env sh

export GITHUB_TOKEN=""
export TEMPLATES_DIR=""
export NEWRELIC_APPNAME=""
export NEWRELIC_LICENSEKEY=""
export PORT=
export TZ="America/Bahia"

GOOS=linux GOARCH=amd64 go build -o app cmd/main.go
