# RSS-to-Static-Site Pipeline (RSS2Hugo)

## 1. Stack Tecnológico
- **Ingesta:** Go (Binario compilado).
- **Generador:** Hugo (Static Site Generator).
- **Servidor Web:** Nginx.
- **SO:** Ubuntu Server / Debian.

## 2. Flujo de Arquitectura
1. **Fetcher (Go):** - Consume 30 feeds RSS.
   - Parsea XML y normaliza a Markdown.
   - Genera archivos en `./content/posts/`.
2. **SSG (Hugo):**
   - Detecta nuevos archivos `.md`.
   - Compila el sitio usando el Theme seleccionado.
   - Minifica activos (HTML/CSS/JS).
3. **WebServer (Nginx):**
   - Sirve contenido estático desde `/var/www/html/`.

## 3. Estructura del Proyecto
├── bin/                # Ejecutable del fetcher en Go
├── src/                # Código fuente del scraper
├── hugo-site/          # Directorio raíz de Hugo
│   ├── content/        # Markdown generado automáticamente
│   ├── themes/         # Temas visuales (Cards)
│   └── public/         # Salida final (estáticos)
└── deploy.sh           # Script de orquestación (Cron)