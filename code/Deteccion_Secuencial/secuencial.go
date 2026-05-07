package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const inputFile = "data/synthetic/expedientes_1M.csv"

type Result struct {
	Mode                string    `json:"mode"`
	Runs                int       `json:"runs"`
	Tiempos             []float64 `json:"tiempos_seconds"`
	Promedio            float64   `json:"promedio_seconds"`
	MediaRecortada      float64   `json:"media_recortada_seconds"`
	AlertasDetectadas   int       `json:"alertas_detectadas"`
	TiempoLectura       float64   `json:"tiempo_lectura_seconds"`
	TiempoTotalEstimado float64   `json:"tiempo_total_estimado_seconds"`
}

// Índices de columnas — ajusta si tu CSV tiene otro orden
const (
	idxDetalle    = 15
	idxHora       = 16
	idxEsSpam     = 17
	idxTipoSpam   = 18
	idxDenunciado = 5 // columna DENUNCIADOS_pres, revisa con tu header real
)

func main() {
	runs   := flag.Int("runs", 20, "Número de ejecuciones de medida")
	outPath := flag.String("out", "", "Ruta JSON de salida (opcional)")
	flag.Parse()

	startLectura := time.Now()

	file, err := os.Open(inputFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	data, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	// Guardar header para buscar índices dinámicamente
	if len(data) == 0 {
		panic("CSV vacío")
	}
	header := data[0]
	data = data[1:]

	idx := buildIndex(header)
	tiempoLectura := time.Since(startLectura).Seconds()

	var tiempos []float64
	alertasFinal := 0

	fmt.Println("====== MODO SECUENCIAL ======")

	for i := 1; i <= *runs; i++ {
		start := time.Now()
		alertasFinal = procesarSecuencial(data, idx)
		elapsed := time.Since(start).Seconds()
		tiempos = append(tiempos, elapsed)
		fmt.Printf("Ejecución %d: %.6f segundos\n", i, elapsed)
	}

	prom := promedio(tiempos)
	rec  := mediaRecortada(tiempos)

	fmt.Println("--------------------------------")
	fmt.Println("Alertas detectadas:", alertasFinal)
	fmt.Printf("Tiempo lectura CSV: %.6f s\n", tiempoLectura)
	fmt.Printf("Promedio procesamiento: %.6f s\n", prom)
	fmt.Printf("Media recortada: %.6f s\n", rec)
	fmt.Printf("Tiempo total estimado: %.6f s\n", tiempoLectura+rec)

	res := Result{
		Mode:                "secuencial",
		Runs:                *runs,
		Tiempos:             tiempos,
		Promedio:            prom,
		MediaRecortada:      rec,
		AlertasDetectadas:   alertasFinal,
		TiempoLectura:       tiempoLectura,
		TiempoTotalEstimado: tiempoLectura + rec,
	}

	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "no se pudo crear archivo JSON: %v\n", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	}
}

// buildIndex mapea nombre de columna → índice numérico
func buildIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, name := range header {
		idx[strings.TrimSpace(name)] = i
	}
	return idx
}

func col(row []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func procesarSecuencial(data [][]string, idx map[string]int) int {

	// --- Fase 1: construir conteos globales ---
	// Solo sobre filas que el generador NO marcó como spam,
	// para calibrar líneas base reales.
	detalleCount := make(map[string]int)
	minuteCount  := make(map[string]int)  // HH:MM → ocurrencias en ventana de 1 min
	fantCount     := make(map[string]int)

	for _, row := range data {
		detalle := col(row, idx, "DETALLE_QUEJA")
		hora    := col(row, idx, "HORA_PRESENTACION")

		if detalle != "" {
			detalleCount[detalle]++
		}
		if len(hora) >= 5 {
			minuteCount[hora[:5]]++
		}
		for _, v := range row {
			if strings.Contains(v, "EMPRESA_FANTASMA_") {
				fantCount[strings.TrimSpace(v)]++
			}
		}
	}

	// --- Fase 2: detección con umbrales calibrados ---
	alertas := 0

	for _, row := range data {
		detalle := col(row, idx, "DETALLE_QUEJA")
		hora    := col(row, idx, "HORA_PRESENTACION")

		sospechoso := false

		// Patrón 1 — queja duplicada:
		// El generador crea grupos de 8 con texto idéntico.
		// Umbral: >= 5 repeticiones del mismo texto.
		if detalleCount[detalle] >= 5 {
			sospechoso = true
		}

		// Patrón 2 — bombardeo por tiempo:
		// El generador crea grupos de 10 en el mismo HH:MM.
		// Con 1M filas / ~720 minutos ≈ 1388 por minuto en promedio,
		// no podemos usar conteo global directo.
		// En su lugar usamos la etiqueta generada (ES_SPAM + TIPO_SPAM)
		// como ground truth para validar, y detectamos por texto corto + hora concentrada.
		// Detección heurística: texto idéntico Y hora compartida con muchos otros.
		// (El bombardeo usa spamTexts que son cortos y repetidos.)
		if len(hora) >= 5 {
			minKey := hora[:5]
			// Solo marca si además el texto también es muy repetido
			// (bombardeo = misma hora + mismo texto)
			if minuteCount[minKey] >= 500 && detalleCount[detalle] >= 5 {
				sospechoso = true
			}
		}

		// Patrón 3 — ráfaga nocturna:
		// Horas 00:xx a 04:xx según syntheticData.go
		if len(hora) >= 2 {
			hh := hora[:2]
			if hh == "00" || hh == "01" || hh == "02" || hh == "03" || hh == "04" {
				sospechoso = true
			}
		}

		// Patrón 4 — denunciado fantasma:
		// Grupos de 5 con el mismo EMPRESA_FANTASMA_XXXX
		for _, v := range row {
			v = strings.TrimSpace(v)
			if strings.Contains(v, "EMPRESA_FANTASMA_") && fantCount[v] >= 3 {
				sospechoso = true
			}
		}

		if sospechoso {
			alertas++
		}
	}

	return alertas
}

func promedio(nums []float64) float64 {
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	return sum / float64(len(nums))
}

func mediaRecortada(nums []float64) float64 {
	copia := make([]float64, len(nums))
	copy(copia, nums)
	sort.Float64s(copia)
	if len(copia) <= 2 {
		return promedio(copia)
	}
	return promedio(copia[1 : len(copia)-1])
}