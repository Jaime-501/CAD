package app

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestNucleoPortableSinWailsNiRutasDeUsuario es la guardia ejecutable de
// T37: recorre internal/** (archivos .go que NO son _test) con go/parser y
// falla si el núcleo se ata a Wails o al perfil del usuario, que es lo que
// rompería el port a Android (un .aar con gomobile no trae Wails ni
// os.UserConfigDir).
//
// Reglas que fija: P2 (internal/** no importa github.com/wailsapp/) y P3
// (os.UserConfigDir/os.UserHomeDir solo en app.go, en la raíz: fuera de
// internal/** y por tanto fuera de este barrido). Corre sin red y sin nada
// más que la propia caché de Go.
func TestNucleoPortableSinWailsNiRutasDeUsuario(t *testing.T) {
	raiz := raizModulo(t)
	dirInternal := filepath.Join(raiz, "internal")
	fset := token.NewFileSet()

	violaciones := []string{}
	err := filepath.WalkDir(dirInternal, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(raiz, ruta)
		fu, err := parser.ParseFile(fset, ruta, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("no pude parsear %s: %w", rel, err)
		}
		violaciones = append(violaciones, violacionesDe(fset, rel, fu)...)
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo internal/**: %v", err)
	}
	for _, v := range violaciones {
		t.Error(v)
	}
	t.Logf("internal/** revisado: sin wailsapp ni rutas del usuario")
}

// violacionesDe devuelve las reglas de portabilidad rotas por un archivo.
func violacionesDe(fset *token.FileSet, rel string, fu *ast.File) []string {
	var fuera []string

	for _, imp := range fu.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if strings.HasPrefix(path, "github.com/wailsapp/") {
			fuera = append(fuera, fmt.Sprintf("%s: P2: importa %q (solo app.go/main.go pueden)", rel, path))
		}
	}

	ast.Inspect(fu, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "os" {
			return true
		}
		if sel.Sel.Name == "UserConfigDir" || sel.Sel.Name == "UserHomeDir" {
			linea := fset.Position(sel.Pos()).Line
			fuera = append(fuera, fmt.Sprintf("%s:%d: P3: usa os.%s (solo app.go, en la raíz)", rel, linea, sel.Sel.Name))
		}
		return true
	})
	return fuera
}

// raizModulo sube desde el directorio de trabajo del test (internal/app)
// hasta encontrar go.mod: así el barrido no depende de desde dónde se lance
// `go test`.
func raizModulo(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("no sé el directorio de trabajo: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			t.Fatalf("no encuentro go.mod subiendo de %s", dir)
		}
		dir = padre
	}
}
