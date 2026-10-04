# Reglas comunes del proyecto

Se aplican a TODAS las tareas. Léelas antes de empezar.

## Layout de paquetes (obligatorio)

```text
app.go
internal/
  logx/                     # slog + redacción de secretos           (T01)
  httpx/                    # cliente HTTP, errores, reintentos      (T02)
  models/                   # modelos de dominio                     (T03, T05)
  scraper/                  # SOLO interfaz Provider y registro      (T04)
    jkanime/                # todo lo de JKAnime (paquete jkanime)
      exceptions.go
      search.go
      detail.go
      episodes.go
      episode_page.go
      extractors/           # paquete extractors
    animeflv/               # paquete animeflv                       (T18)
  stream/                   # proxy local de streaming               (T19)
  cache/                    # caché en memoria con TTL               (T21)
  db/                       # historial de visualización             (T22)
frontend/src/
```

Regla de dependencias entre paquetes (sentido único, prohibido el ciclo):

```text
models  ←  httpx  ←  scraper (interfaz)  ←  scraper/jkanime  ←  app.go
                                          ←  scraper/jkanime/extractors
logx ← (todos)
```

- `extractors` no importa `jkanime`; recibe `httpx.HTTPDoer` por parámetro.
- `jkanime` importa `extractors`.
- `models` no importa a nadie del proyecto.

---

## Reglas generales

1. No depender de una única regex para obtener vídeo.
2. Separar extracción de servidores de resolución del recurso final.
3. Mantener cookies, User-Agent y sesión durante todo el flujo.
4. Inyección de dependencias (`httpx.HTTPDoer` + `httptest.Server`) para aislar los tests de la red.
5. El frontend nunca se bloquea esperando peticiones largas (`runtime.EventsEmit`).
6. No tocar `frontend/wailsjs/**` salvo en la tarea de regeneración de bindings (T23).
7. **Regla de escape:** si necesitas modificar un archivo fuera de la lista "Puedes tocar", DETENTE y repórtalo (archivo, motivo, cambio propuesto). No improvises.
8. **Puerta de calidad global** (se cumple en cada tarea que toque Go): `gofmt -l .` sin salida, `go vet ./...`, `go test ./...`. En tareas con concurrencia (marcadas con 🔒) añade `go test -race ./...`.
9. Los enlaces de vídeo son temporales: **nunca** se cachean ni se persisten.
10. Nunca se registran en logs cookies, tokens CSRF ni cabeceras `Authorization`/`Set-Cookie`.
