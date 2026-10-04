# Dockerfile
FROM golang:1.25-bookworm

# 1. Instalar dependencias del sistema (GTK, WebKit) y Xvfb (Pantalla Virtual)
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libgtk-3-dev \
    libwebkit2gtk-4.0-dev \
    xvfb \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 2. Instalar Node.js (Requerido para el frontend Vite/Svelte)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs

# 3. Instalar la CLI de Wails
RUN go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Asegurar que los binarios de Go estén en el PATH
ENV PATH=$PATH:/go/bin

WORKDIR /app
