# Changelog

## 1.4.0

Rediseño y enfoque móvil de la interfaz. La app ahora se adapta a ventanas
estrechas (móvil y tableta) con una navegación inferior, y **el escritorio
conserva su geometría**: las tarjetas, la rejilla de episodios, la barra lateral
y la cabecera se mantienen igual por encima de 900 px de ancho.

### Añadido

- **Barra de navegación inferior en modo estrecho** (T73): en ventanas de menos de 900 px desaparece la hamburguesa y aparece una barra fija con cuatro destinos (Descubrir, Buscar, Favoritos e Historial) que reutiliza el cajón de historial. La barra marca el destino activo con `aria-current` y un indicador superior, y respeta el área segura del dispositivo.
- **Cabecera compacta de detalle y reproductor** (T74): un único componente con «Atrás», título de una línea con elipsis y la estrella de favoritos; en móvil la cabecera del detalle queda fija al bajar.
- **Fila «Continuar viendo» en el inicio móvil** (T77): las últimas series vistas, con portada, progreso y acceso directo al episodio pendiente.
- **Ficha de detalle con sinopsis plegable** (T79): portada enmarcada, metadatos en insignias, géneros en una línea con desplazamiento y un botón «Ver más» para leer la sinopsis completa en pantallas estrechas.
- **Barra de acción fija «Continuar / Reiniciar»** (T80): en móvil, los accesos al episodio se quedan pegados encima de la barra inferior, siempre al alcance del pulgar.
- **Orden de episodios ascendente/descendente** (T82): un botón invierte la presentación de la rejilla de episodios sin alterar la carga ni el progreso.
- **Sistema de tokens y primitivas de interfaz** (T70, T71, T72): paleta esmeralda, escala tipográfica, tipografías locales **Inter** (texto) y **Outfit** (títulos) por subconjunto latino, espacio en múltiplos de 4, objetivos táctiles de 44 px, conjunto de iconos y componentes base (botón, chip, insignia, esqueleto y estado vacío/error/cargando).
- **Estados unificados** (T86): inicio, detalle, reproductor y cajón comparten los mismos estados de carga, error y vacío, con aviso de reintento.
- **Accesibilidad AA** (T88): una sola cabecera de nivel 1 por vista sin saltos de jerarquía, nombres accesibles en todos los controles, contraste verificado en texto normal, con foco y en hover, anillo de foco visible y `aria-busy` en los estados de carga.
- **Trampa de foco del cajón** (T89): con el historial abierto en móvil, el contenido de fondo y la barra inferior quedan inertes, así que el teclado no sale del cajón.

### Cambiado

- **Paleta esmeralda** en toda la app: se retira la paleta gris anterior y los colores, radios y sombras pasan aTokens del nuevo sistema.
- **Tipografías locales Inter y Outfit** (subconjunto latino, sin peticiones a Google Fonts) en lugar de Nunito.
- **Rejilla de resultados de búsqueda** (T78): dos columnas en móvil, tres en tableta y rejilla automática en escritorio, con contador de resultados y esqueletos sin salto de layout.
- **Tarjeta de episodio** (T81): miniatura 16:9, insignia de estado, franja de progreso y altura constante por rango de columnas.
- **Chip «Siguiente ▸» del reproductor** (T84): pasa a objetivo táctil de 44 px y se recoloca para no tapar el vídeo.

### Corregido

- **Fondo dividido** (T71): la franja azul que aparecía al final de la página desaparece; el fondo es uniforme en toda la ventana.
- **Cajón opaco** (T73b): el historial ya no deja traslucir el contenido ni el pie «Borrar todo».
- **Idioma de la página** (T89): `<html lang="es">` (antes declaraba inglés).
- **Foco del cajón** (T89): el teclado ya no puede salir del cajón abierto hacia la barra inferior o el contenido de fondo.

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
