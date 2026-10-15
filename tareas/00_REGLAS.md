# Reglas comunes del proyecto

Se aplican a TODAS las tareas. Léelas antes de empezar.

## Layout de paquetes (obligatorio)

```text
main.go                   entrada Wails; incrusta frontend/dist (//go:embed)
app.go                    capa Wails: traduce llamadas ↔ núcleo           (T37)
internal/
  app/                    núcleo: orquestación testeable, sin Wails       (T37)
    nucleo.go             registro, proxy, caché, historial, favoritos, Home
    retomar.go            "Retomar" y "Siguiente ▸"                       (T46)
    portabilidad_test.go  guardia P2/P3 con go/parser
    app_e2e_test.go       E2E offline (flujo completo / AnimeAV1)
    *_integration_test.go  E2E y smoke con red real (tag integration)
  logx/                   # slog + redacción de secretos                  (T01)
    logx.go
  httpx/                  # cliente HTTP, errores, reintentos             (T02)
    httpx.go  errors.go  retry.go
  models/                 # modelos de dominio                            (T03, T05)
    models.go  ids.go  v12.go
  scraper/                # SOLO interfaz Provider y registro             (T04)
    provider.go  registry.go
    animeav1/             # todo lo de AnimeAV1 (paquete animeav1)
      animeav1.go  detail.go  devalue.go  episodes.go  home.go  sources.go
      testdata/           # fixtures __data.json y HTML crudos
    jkanime/              # todo lo de JKAnime (paquete jkanime)
      jkanime.go  search.go  detail.go  episodes.go  episode_page.go
      exceptions.go  home.go
      testdata/           # fixtures HTML/JSON crudos
      extractors/         # paquete extractors
        extractor.go  jkplayer.go  jkphp.go  servers.go
        mediafire.go  normalize.go  selector.go
        testdata/         # fixtures HTML de servidores
  stream/                 # proxy local de streaming                      (T19)
    stream.go  manifiesto.go  reenvio.go  sesiones.go  red.go
    testdata/             # manifiestos .m3u8 de prueba
  cache/                  # caché en memoria con TTL                      (T21)
    cache.go
  db/                     # historial y favoritos (JSON en disco)         (T22, T42)
    db.go  favoritos.go
docs/
  firmas/                 # firmas reales por paquete (httpx.md, models.md, ...)
  investigacion/          # notas de investigación (animeav1.md, home.md)
  img/                    # capturas para la documentación
  BITACORA.md             # historial; NO se lee por defecto
  DESARROLLO.md           # guía de desarrollo
  PROGRESO_v1.0.md  PROGRESO_v1.1.md  PROGRESO_v1.2.md  # historiales por versión
frontend/
  index.html              # documento HTML (meta referrer no-referrer)
  src/
    api.ts                # ÚNICO punto de acceso al backend (P4)
    stores.ts  main.ts  views.ts  ui.ts  style.css
    favoritos.ts  historial.ts  home.ts  retomar.ts  toasts.ts
    etiquetas.ts  deslizar.ts  vite-env.d.ts  (+ *.test.ts)
    App.svelte  Vista*.svelte  Tarjeta*.svelte  Boton*.svelte
    BarraLateral.svelte  FilaAnimes.svelte  GridFavoritos.svelte
    SelectorFuente.svelte  Toast.svelte
    assets/               # fuentes y logo
  wailsjs/                # bindings generados por Wails — NO EDITAR
```

Regla de dependencias entre paquetes (sentido único, prohibido el ciclo):

```text
models  ←  httpx  ←  extractors          ←  scraper/jkanime  ←  internal/app  ←  app.go
                  ←  scraper (interfaz)  ←  scraper/jkanime
                                           ←  scraper/animeav1
logx ← (todos)
```

- `extractors` no importa `jkanime`; recibe `httpx.HTTPDoer` por parámetro.
- `jkanime` importa `extractors`; `animeav1` **no** importa `jkanime` (ni `extractors`).
- `models` no importa a nadie del proyecto.
- `internal/app` (el núcleo) importa scraper, `scraper/jkanime`, `scraper/animeav1`, `extractors`, cache, db, httpx, models y stream; **no** importa `github.com/wailsapp/*` (P2). `app.go` es la única capa que habla con Wails.

---

## Reglas de portabilidad (T37, aplican a todas las tareas futuras)

1. **P1.** La lógica nueva va en `internal/**`, nunca en `package main`.
2. **P2.** `internal/**` no importa `github.com/wailsapp/*`.
3. **P3.** `os.UserConfigDir`/`os.UserHomeDir` solo se llaman en `app.go` (único punto); el núcleo recibe el directorio de datos por parámetro.
4. **P4.** El frontend habla con el backend solo a través de `frontend/src/api.ts` (ningún otro archivo importa `wailsjs`).
5. **P5.** El vídeo se entrega siempre como `VideoSource` (con `Headers`/`Referer`); el frontend no asume que exista el proxy.
6. **P6.** UI nueva sin depender de hover ni de anchos fijos.
7. **P7.** Funciones solo de escritorio (ventana, bandeja, diálogos) viven en `main.go`/`app.go`.

P2 y P3 tienen guardia ejecutable: `internal/app/portabilidad_test.go` recorre `internal/**` con `go/parser` y falla si aparece un import de `github.com/wailsapp/` o una llamada a `os.UserConfigDir`/`os.UserHomeDir`.

---

## Reglas generales

1. No depender de una única regex para obtener vídeo.
2. Separar extracción de servidores de resolución del recurso final.
3. Mantener cookies, User-Agent y sesión durante todo el flujo.
4. Inyección de dependencias (`httpx.HTTPDoer` + `httptest.Server`) para aislar los tests de la red.
5. El frontend nunca se bloquea esperando peticiones largas (`runtime.EventsEmit`).
6. No tocar `frontend/wailsjs/**` salvo en las tareas de regeneración de bindings (T23, T47).
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
