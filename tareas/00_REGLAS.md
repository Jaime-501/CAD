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
    animeav1/               #todo lo de AnimeAv1
    jkanime/                # todo lo de JKAnime (paquete jkanime)
      exceptions.go
      search.go
      detail.go
      episodes.go
      episode_page.go
      extractors/           # paquete extractors
        extractor.go
        jkanime.go
        jkphp.go
    animeflv/               # paquete animeflv                       (T18)
  stream/                   # proxy local de streaming               (T19)
  cache/                    # caché en memoria con TTL               (T21)
  db/                       # historial de visualización             (T22)
docs/
  firmas/                   # firmas reales por paquete (httpx.md, models.md, ...)
  BITACORA.md               # historial; NO se lee por defecto
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

---

## Documentación y estado

- `PROGRESO.md` es lo único que se lee siempre: estado de tareas, pendientes activos y desviaciones que afectan a tareas futuras.
- Las **firmas reales** viven en `docs/firmas/<paquete>.md`. Lee solo las de los paquetes de los que depende tu tarea. Donde un archivo de tarea diga "firmas registradas en PROGRESO.md", entiende `docs/firmas/`.
- Si una firma anotada no coincide con el código, **manda el código** (`go doc ./internal/<paquete>`).
- `docs/BITACORA.md` no se lee por defecto; solo se consulta si necesitas el historial de una decisión.
- Actualiza `PROGRESO.md` y `docs/firmas/` en cada hito: al terminar fixtures, implementación y tests.
- Al cerrar la tarea, mueve a `docs/BITACORA.md` lo que ya no afecte a tareas futuras.

## Fixtures y pruebas

11. **Nunca inventes HTML ni selectores.** Los parsers se escriben contra fixtures reales en `testdata/`. Si no existe un fixture real, detente y repórtalo. Si debes usar HTML artificial (por ejemplo, un caso que el sitio no ofrece), indícalo en las desviaciones de `PROGRESO.md`.
12. Los fixtures son la respuesta **cruda** del servidor, no el DOM renderizado. Se sanean cookies, tokens CSRF y parámetros firmados de URLs. No se descargan vídeos.
13. Los tests normales nunca usan la red: usan `httptest.Server` con fixtures. Los smoke tests con red van en un archivo aparte `*_integration_test.go` con `//go:build integration` (permitido aunque no figure en "Puedes tocar").
14. Lee solo tu tarea, sus dependencias y las firmas de esos paquetes. Consulta tareas posteriores solo para definir una firma que ellas consumirán; no adelantes su implementación.
15. Si sale de tu alcance un cambio mínimo y necesario (por ejemplo, registrar un extractor), repórtalo en lugar de editarlo en silencio, salvo que el prompt lo autorice.
