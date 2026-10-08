package animeav1

import (
	"encoding/json"
	"fmt"
	"math"
)

// Constantes del formato devalue (fuente: devalue/src/constants.js). Dentro
// de un objeto o de un array, un número nunca es un valor: es la referencia
// por índice a otro elemento del mismo array plano. Los números negativos no
// son índices, sino valores especiales:
//
//	-1  undefined     → nil
//	-2  hueco de array → nil (conserva la longitud)
//	-3  NaN           → math.NaN()
//	-4  Infinity      → math.Inf(1)
//	-5  -Infinity     → math.Inf(-1)
//	-6  -0            → math.Copysign(0, -1)
//	-7  array disperso (sparse) → NO soportado: error
const (
	devalueUndefined   = -1
	devalueHueco       = -2
	devalueNaN         = -3
	devalueInfPositiva = -4
	devalueInfNegativa = -5
	devalueCeroNeg     = -6
	devalueSparse      = -7
)

// Unflatten convierte el array plano de un nodo devalue (el campo data de un
// __data.json de SvelteKit) en los valores Go correspondientes, resolviendo
// las referencias por índice. El elemento 0 del array es la raíz.
//
// Soporta: null, booleanos, cadenas y números literales (los que van sueltos
// en el array), objetos, arrays, undefined, huecos (los dos últimos → nil) y
// los valores especiales numéricos NaN, ±Inf y -0. Devuelve error ante tipos
// desconocidos: arrays con etiqueta (Date, Map, Set, RegExp, Object, BigInt,
// arrays tipados, "null"), arrays dispersos (−7), un slot que no sea número
// o una referencia fuera de rango. Nunca hace panic.
//
// Los ciclos (un objeto que referencia a un ancestro) se resuelven porque
// cada contenedor se publica en valores[] antes de rellenarse, igual que
// hace devalue.
func Unflatten(plano []any) (any, error) {
	if len(plano) == 0 {
		return nil, fmt.Errorf("devalue: el array está vacío")
	}
	valores := make([]any, len(plano))
	resuelto := make([]bool, len(plano))

	var materializar func(i int) (any, error)

	hidratar := func(ref float64) (any, error) {
		switch ref {
		case devalueUndefined, devalueHueco:
			return nil, nil
		case devalueNaN:
			return math.NaN(), nil
		case devalueInfPositiva:
			return math.Inf(1), nil
		case devalueInfNegativa:
			return math.Inf(-1), nil
		case devalueCeroNeg:
			return math.Copysign(0, -1), nil
		}
		if ref < 0 {
			return nil, fmt.Errorf("devalue: constante desconocida %v", ref)
		}
		if ref != math.Trunc(ref) {
			return nil, fmt.Errorf("devalue: la referencia %v no es un índice entero", ref)
		}
		i := int(ref)
		if i >= len(plano) {
			return nil, fmt.Errorf("devalue: la referencia %d está fuera del array (%d valores)", i, len(plano))
		}
		return materializar(i)
	}

	materializar = func(i int) (any, error) {
		if resuelto[i] {
			return valores[i], nil
		}
		switch v := plano[i].(type) {
		case nil:
			resuelto[i] = true
			return nil, nil
		case bool:
			valores[i], resuelto[i] = v, true
			return v, nil
		case string:
			valores[i], resuelto[i] = v, true
			return v, nil
		case []any:
			if len(v) > 0 {
				if etiqueta, ok := v[0].(string); ok {
					return nil, fmt.Errorf("devalue: tipo con etiqueta no soportado %q en la posición %d", etiqueta, i)
				}
				if num, ok := numeroDe(v[0]); ok && num == devalueSparse {
					return nil, fmt.Errorf("devalue: array disperso (sparse) no soportado en la posición %d", i)
				}
			}
			res := make([]any, len(v))
			valores[i], resuelto[i] = res, true
			for k, elem := range v {
				ref, ok := numeroDe(elem)
				if !ok {
					return nil, fmt.Errorf("devalue: el elemento %d del array de la posición %d no es una referencia (%T)", k, i, elem)
				}
				if ref == devalueHueco {
					continue // hueco: queda nil y se conserva la longitud
				}
				val, err := hidratar(ref)
				if err != nil {
					return nil, err
				}
				res[k] = val
			}
			return res, nil
		case map[string]any:
			res := make(map[string]any, len(v))
			valores[i], resuelto[i] = res, true
			for k, elem := range v {
				ref, ok := numeroDe(elem)
				if !ok {
					return nil, fmt.Errorf("devalue: el campo %q del objeto de la posición %d no es una referencia (%T)", k, i, elem)
				}
				val, err := hidratar(ref)
				if err != nil {
					return nil, err
				}
				res[k] = val
			}
			return res, nil
		default:
			if num, ok := numeroDe(v); ok {
				valores[i], resuelto[i] = num, true
				return num, nil
			}
			return nil, fmt.Errorf("devalue: tipo desconocido %T en la posición %d", v, i)
		}
	}

	return hidratar(0)
}

// PageData desempaqueta el cuerpo de un __data.json de SvelteKit y devuelve
// los datos del nodo de página, que es el último nodo con type "data" (los
// anteriores son layouts compartidos, p. ej. el nodo auth).
//
// Tolera nodos null y nodos con type "skip"/"error": se ignoran. Un nodo
// data mal formado, una respuesta con otro type y un JSON inválido son error;
// tampoco hace panic.
func PageData(crudo []byte) (map[string]any, error) {
	var doc struct {
		Type  string `json:"type"`
		Nodes []any  `json:"nodes"`
	}
	if err := json.Unmarshal(crudo, &doc); err != nil {
		return nil, fmt.Errorf("devalue: el cuerpo no es JSON válido: %w", err)
	}
	if doc.Type != "data" {
		return nil, fmt.Errorf("devalue: type = %q, quería \"data\"", doc.Type)
	}
	for i := len(doc.Nodes) - 1; i >= 0; i-- {
		nodo, ok := doc.Nodes[i].(map[string]any)
		if !ok {
			continue // nodo null u otro tipo de escalar
		}
		tipo, _ := nodo["type"].(string)
		if tipo != "data" {
			continue // "skip", "error" u otros: no llevan datos de página
		}
		plano, ok := nodo["data"].([]any)
		if !ok {
			return nil, fmt.Errorf("devalue: el nodo de página no trae un array data (%T)", nodo["data"])
		}
		datos, err := Unflatten(plano)
		if err != nil {
			return nil, err
		}
		obj, ok := datos.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("devalue: los datos del nodo de página no son un objeto (%T)", datos)
		}
		return obj, nil
	}
	return nil, fmt.Errorf("devalue: la respuesta no trae nodo de página")
}

// numeroDe lee el valor de un slot o de un literal numérico. encoding/json
// decodifica los números como float64; el int se acepta también para poder
// construir arrays a mano en los tests.
func numeroDe(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	}
	return 0, false
}
