#!/bin/bash
set -e

echo "=== Micomicona Deploy Script ==="
echo "Fecha: $(date)"

# Directorios
PROJECT_ROOT=$(pwd)
SRC_DIR="$PROJECT_ROOT/src"
BIN_DIR="$PROJECT_ROOT/bin"
HUGO_DIR="$PROJECT_ROOT/hugo-site"

# 1. Compilar y Ejecutar Fetcher
echo "-> Compilando Fetcher (Go)..."
cd "$SRC_DIR"
go build -o "$BIN_DIR/micomicona-fetcher"
echo "-> Ejecutando Fetcher..."
cd "$PROJECT_ROOT"
"$BIN_DIR/micomicona-fetcher"

# 2. Compilar sitio Hugo
echo "-> Compilando sitio estático (Hugo)..."
cd "$HUGO_DIR"
# Usamos --minify para optimizar HTML/CSS/JS
hugo --minify

echo "-> ¡Despliegue finalizado exitosamente!"
# Los archivos listos para Nginx están en hugo-site/public/
