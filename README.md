

Readme · MD
# Comfort Anime Desktop
 
Aplicación de escritorio para Windows que te permite **buscar anime, ver la lista de episodios y reproducirlos** en una interfaz sencilla, con tu progreso guardado para retomar donde lo dejaste.
 
![Búsqueda](docs/img/Busqueda.png)

![Detalles](docs/img/Detalles.png)

![Reproductor](docs/img/Reproduccion.png)
 
## Qué puedes hacer
 
- **Buscar** anime por nombre.
- **Ver el detalle y los episodios** de cada serie (la lista aparece poco a poco mientras carga).
- **Reproducir** el episodio directamente en la app, con selector de servidor/calidad si el episodio tiene varias fuentes, y subtítulos cuando la fuente los incluye.
- **Retomar donde lo dejaste:** la app recuerda el minuto de cada episodio, marca los vistos y tiene un botón **Continuar**.
## Instalación
 
1. Ve a la sección [**Releases**](../../releases) de este repositorio y descarga el archivo `.zip` de la última versión.
2. Descomprímelo en la carpeta que quieras.
3. Ejecuta `miapp.exe`. No necesita instalador.
**Requisitos:** Windows 11 (usa el componente WebView2, que ya viene incluido). En Windows 10 puede que tengas que instalar [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) por tu cuenta.
 
> **Aviso de Windows (SmartScreen):** como el ejecutable no está firmado digitalmente, Windows puede mostrar "Windows protegió su PC". Pulsa **Más información → Ejecutar de todas formas**. Si quieres verificar que el archivo es el original, compara su suma SHA256 con la que aparece en las notas de la release (`Get-FileHash archivo.zip` en PowerShell).
 
## Cómo se usa
 
1. Escribe el nombre de un anime en el buscador (los resultados aparecen solos).
2. Haz clic en una serie para ver sus episodios.
3. Haz clic en un episodio para reproducirlo. Si un servidor no carga, elige otro en el selector.
4. Vuelve atrás cuando quieras: tu progreso se guarda automáticamente cada pocos segundos.
## Tus datos
 
La app solo guarda tu historial de visualización, en tu propio equipo:
 
- `%AppData%\miapp\historial.json` — el progreso de cada episodio. Puedes copiarlo para hacer una copia de seguridad o borrarlo para empezar de cero.
- `%AppData%\miapp.exe\` — carpeta del motor de la interfaz (WebView2). Se crea sola; puedes ignorarla.
No hay cuentas ni registro, y la app no envía tus datos a ningún servidor propio.
 
## Problemas frecuentes
 
| Problema | Qué hacer |
|---|---|
| **No aparecen resultados** | Prueba con otro nombre o espera un momento: los sitios de origen pueden estar caídos o haber cambiado. |
| **El episodio no carga** | Elige otro servidor en el selector. Si ninguno funciona, vuelve a intentarlo más tarde. |
| **Un servidor aparece como "iframe"** | Es un reproductor externo incrustado: funciona, pero **no guarda tu progreso**. |
| **AnimeFLV no muestra detalle** | Es normal: AnimeFLV solo se usa como respaldo para la búsqueda. |
| **"Continuar" no abre el último que vi** | Elige el episodio de mayor número con progreso, no el más reciente. Es una limitación conocida. |
 
## Limitaciones conocidas
 
- Solo para **Windows**.
- **JKAnime** es el proveedor principal. **AnimeFLV** se usa únicamente como respaldo de búsqueda (sin episodios ni vídeo).
- Si un sitio cambia su estructura, la búsqueda o la reproducción pueden dejar de funcionar hasta que se actualice la app.
- No hay opción para borrar entradas individuales del historial.
## Aviso legal
 
Este es un proyecto **personal y educativo**. La aplicación **no aloja, almacena ni distribuye ningún contenido audiovisual**: solo consulta sitios web de terceros y muestra lo que esos sitios ofrecen. Eres responsable de usarla conforme a las leyes de tu país y a los términos de los sitios que consultes. Los nombres y las marcas pertenecen a sus respectivos dueños, y este proyecto no está afiliado a ninguno de ellos.
 
## Para desarrolladores
 
Compilar desde el código, estructura del proyecto, tests y cómo publicar versiones: consulta [`docs/DESARROLLO.md`](docs/DESARROLLO.md).
