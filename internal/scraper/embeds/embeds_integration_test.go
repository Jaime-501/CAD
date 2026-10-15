//go:build integration

package embeds

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// envEmbedMP4Upload es la variable con la URL de un embed real de MP4Upload.
// El id no se pega en el repositorio ni en PROGRESO.md: sale de un episodio
// real vigente que se pasa en el entorno al lanzar el smoke.
const envEmbedMP4Upload = "MIAPP_MP4UPLOAD_EMBED"

// Smoke de red: resuelve un embed real de MP4Upload y confirma contra el CDN
// con Range: bytes=0-0 que responde 206 (sin descargar vídeo). Requiere
// conectividad y un embed vivo en MIAPP_MP4UPLOAD_EMBED; sin él se salta.
//
//	MIAPP_MP4UPLOAD_EMBED=https://www.mp4upload.com/embed-XXXXXXXXXXXX.html \
//	  go test -tags integration ./internal/scraper/embeds/ -v
func TestMP4UploadRedReal(t *testing.T) {
	embed := strings.TrimSpace(os.Getenv(envEmbedMP4Upload))
	if embed == "" {
		t.Skipf("define %s con la URL de un embed real de MP4Upload para ejecutar este smoke", envEmbedMP4Upload)
	}
	cli := httpx.NewClient(httpx.ClientOptions{})
	r := NewMP4Upload(cli)
	if !r.CanHandle(embed) {
		t.Fatalf("CanHandle(%q) = false", embed)
	}

	fuentes, err := r.Resolve(context.Background(), embed, "")
	if err != nil {
		t.Fatalf("Resolve falló: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(fuentes))
	}
	f := fuentes[0]
	if f.Type != models.ResourceMP4 || f.Server != nombreMP4Upload || f.URL == "" {
		t.Fatalf("fuente inesperada: type=%q server=%q url=%s", f.Type, f.Server, urlCorta(f.URL))
	}

	// Confirmación del CDN: Range: bytes=0-0 (206, sin descargar vídeo) con
	// el Referer de la fuente. El CDN responde 403 sin un Referer del
	// dominio mp4upload.com (verificado en vivo; desviación del informe).
	resp, err := httpx.Get(context.Background(), cli, f.URL, httpx.ReqOptions{
		Provider:  nombreMP4Upload,
		Operation: "mp4upload",
		Referer:   f.Referer,
		Header:    http.Header{"Range": []string{"bytes=0-0"}},
	})
	if err != nil {
		t.Fatalf("el CDN no respondió a Range: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("status del CDN = %d, quería 206", resp.StatusCode)
	}
	cabecera := resp.Header.Get("Content-Range")
	if cabecera == "" {
		t.Error("el CDN no mandó Content-Range")
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1))
	t.Logf("MP4 resuelto: %s | Range 206 | Content-Range: %s", urlCorta(f.URL), cabecera)
}

// urlCorta deja la URL del medio legible sin pegar el token del path
// (/d/<token>/video.mp4).
func urlCorta(u string) string {
	parte := strings.Split(u, "/")
	if len(parte) >= 2 && len(parte[len(parte)-2]) > 8 {
		parte[len(parte)-2] = parte[len(parte)-2][:4] + "…"
	}
	return strings.Join(parte, "/")
}
