# Changelog

## 1.3.0

### Añadido

- **Revelado animado de las tarjetas de episodio** (T65): al entrar en el viewport, cada tarjeta aparece con una animación breve de opacidad y desplazamiento, escalonada por columna de la fila; respeta `prefers-reduced-motion: reduce` (sin animación) y no se repite al volver del reproductor. La lógica de retardos vive en `ventana.ts` y la acción de revelado en `revelar.ts`.

### Corregido

- **Scroll del detalle separado por proveedor + slug** (T66): la clave de scroll del detalle y del reproductor incluye ahora el proveedor, así que «bleach» en JKAnime y en AnimeAV1 ya no comparten posición de desplazamiento.
- **Restauración del scroll del detalle al recargar la lista** (T67): al volver a un detalle cuyos episodios se recargan por lotes, la posición guardada se restaura en cuanto la lista alcanza la altura necesaria (y se cancela si el usuario hace un gesto), en vez de quedarse arriba.
- **Altura reservada de las tarjetas de episodio** (T65): `contain-intrinsic-size` pasa de `auto 210px` a `auto 170px` (la altura real medida), eliminando el salto de scroll al entrar tarjetas fuera de pantalla.

## 1.2.1

### Añadido

- **Botón "Siguiente ▸" discreto durante la reproducción** (T57): en el reproductor, cuando el siguiente episodio está listo el botón queda translúcido y desenfocado hasta que se le pasa el ratón o recibe el foco con el teclado; nunca es discreto al llegar al 90 % del episodio (resaltado) y respeta `prefers-reduced-motion`. (La mejora no figuraba en el CHANGELOG: la 1.2.0 no tenía entrada.)

### Corregido

- **Carátula del historial al retomar** (T58): al retomar un anime desde la barra lateral mientras el detalle aún cargaba (o con otro anime abierto), la tarjeta del historial ya no perdía la portada: título y portada se toman en cascada (detalle actual del mismo anime → registro en memoria → vacío).
- **Contador de favoritos al arrancar** (T58): "★ Favoritos (N)" muestra el recuento real desde el arranque de la app, sin esperar a abrir la pestaña (con un único reintento si el backend aún no respondía).

## 1.2.0

### Añadido

- **Nombre visible "CAD · Comfort Anime Desktop"** en la ventana (el ejecutable sigue siendo `miapp.exe`).
- **Inicio "Descubrir"** por proveedor (JKAnime y AnimeAV1): filas de "En emisión" y de novedades con carrusel, botón "Actualizar" y "Reintentar" si un proveedor falla.
- **Barra lateral "Continuar viendo"** con el historial agrupado por proveedor, ordenado por recencia, con barra de progreso. Un clic retoma el episodio donde lo dejaste (o abre el siguiente si ya lo viste).
- **Borrado del historial**: ✕ por anime, deslizar, borrar por proveedor y borrar todo, con aviso "Deshacer".
- **Favoritos**: estrella en el detalle y pestaña "★ Favoritos" con cuadrícula (se guardan en `favoritos.json`).
- **Botón "Siguiente ▸"** en el reproductor (también con la tecla N), resaltado al llegar al 90 %.

### Cambiado

- El historial guarda ahora proveedor, título, portada, número de episodio y fecha (`updated_at`). Los `historial.json` anteriores siguen cargando.
- 
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
