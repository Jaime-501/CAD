# Changelog

## 1.2.1

### Añadido

- **Botón "Siguiente ▸" discreto durante la reproducción** (T57): en el reproductor, cuando el siguiente episodio está listo el botón queda translúcido y desenfocado hasta que se le pasa el ratón o recibe el foco con el teclado; nunca es discreto al llegar al 90 % del episodio (resaltado) y respeta `prefers-reduced-motion`. (La mejora no figuraba en el CHANGELOG: la 1.2.0 no tenía entrada.)

### Corregido

- **Carátula del historial al retomar** (T58): al retomar un anime desde la barra lateral mientras el detalle aún cargaba (o con otro anime abierto), la tarjeta del historial ya no perdía la portada: título y portada se toman en cascada (detalle actual del mismo anime → registro en memoria → vacío).
- **Contador de favoritos al arrancar** (T58): "★ Favoritos (N)" muestra el recuento real desde el arranque de la app, sin esperar a abrir la pestaña (con un único reintento si el backend aún no respondía).

## 1.1.0

### Añadido

- **AnimeAV1 como segundo proveedor completo** (búsqueda, detalle, episodios y fuentes de vídeo), junto a JKAnime. El sitio SvelteKit se lee por `__data.json` y los episodios llegan por lotes de 50.
- **Búsqueda combinada**: JKAnime y AnimeAV1 se consultan en paralelo; los resultados de JKAnime van primero y un proveedor caído no deja sin respuesta.
- **Selector de idioma SUB/DUB** en las fuentes de vídeo de AnimeAV1 (`language` en las fuentes, etiquetas del selector `Servidor · Idioma · Calidad`).
- **Reproducción HLS de AnimeAV1** vía proxy local: el manifiesto se deriva sin peticiones extra y el proxy inyecta la cabecera `Sec-Fetch-Site: same-origin` que los segmentos exigen.
- E2E de AnimeAV1 (offline y con red) que demuestran el flujo completo: búsqueda combinada → detalle por lotes → fuentes → manifiesto y segmento por el proxy.

### Corregido

- **Aislamiento del progreso por proveedor**: como `bleach` existe en los dos proveedores, la ficha, los episodios y el progreso (`Continuar`) se filtran por el prefijo del episodio (`<proveedor>:`); abrir el mismo slug en JKAnime y en AnimeAV1 ya no reutiliza la ficha ni el progreso del otro (bug encontrado y corregido en T34).

### Eliminado

- **AnimeFLV retirado**: proveedor de solo búsqueda eliminado del registro, del código y de la documentación. El sentinel genérico `scraper.ErrNoDetails` se conserva para futuros proveedores.
