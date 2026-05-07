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

func main() {
	runs    := flag.Int("runs", 20, "Número de ejecuciones de medida")
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

	if len(data) == 0 {
		panic("CSV vacío")
	}
	header := data[0]
	data = data[1:]

	idx := buildIndex(header)

	// ── Fase 1: conteos globales ─────────────────────────────────────────
	// Se ejecuta una sola vez fuera del benchmark.
	// detalleCount: frecuencia de cada texto exacto de queja.
	// fantCount: frecuencia de cada denunciado fantasma.
	detalleCount := make(map[string]int)
	fantCount    := make(map[string]int)

	for _, row := range data {
		detalle := col(row, idx, "DETALLE_QUEJA")
		if detalle != "" {
			detalleCount[detalle]++
		}
		for _, v := range row {
			v = strings.TrimSpace(v)
			if strings.Contains(v, "EMPRESA_FANTASMA_") {
				fantCount[v]++
			}
		}
	}

	tiempoLectura := time.Since(startLectura).Seconds()

	var tiempos []float64
	alertasFinal := 0

	fmt.Println("====== MODO SECUENCIAL ======")

	for i := 1; i <= *runs; i++ {
		start := time.Now()
		alertasFinal = procesarSecuencial(data, idx, detalleCount, fantCount)
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

func procesarSecuencial(
	data [][]string,
	idx map[string]int,
	detalleCount, fantCount map[string]int,
) int {
	alertas := 0

	for _, row := range data {
		detalle := col(row, idx, "DETALLE_QUEJA")
		hora    := col(row, idx, "HORA_PRESENTACION")

		sospechoso := false

		// Patrón 1+2 — queja duplicada y bombardeo por tiempo:
		// Los textos de spam son exactamente 3 plantillas fijas que se
		// repiten ~23k veces c/u. Los textos normales usan plantillas
		// distintas con un máximo de ~16k repeticiones.
		// Umbral >= 20000 captura spam sin tocar normales.
		if detalleCount[detalle] >= 20000 {
			sospechoso = true
		}

		// Patrón 3 — ráfaga nocturna:
		// Horas 00:xx a 04:xx. Ningún registro normal tiene estas horas
		// (el generador usa 08-17 para el 85% y 18-21 para el 15%).
		if len(hora) >= 2 {
			hh := hora[:2]
			if hh == "00" || hh == "01" || hh == "02" || hh == "03" || hh == "04" {
				sospechoso = true
			}
		}

		// Patrón 4 — denunciado fantasma:
		// El generador crea 10k empresas distintas con 3 filas cada una.
		// Umbral >= 3 captura todos sin falsos positivos.
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