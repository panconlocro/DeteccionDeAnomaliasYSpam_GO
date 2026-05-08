// concurrente.go
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const inputFile = "data/synthetic/expedientes_1M.csv"

var (
	alertasGlobal int
	mutex         sync.Mutex
)

type Result struct {
	Mode                string    `json:"mode"`
	Workers             int       `json:"workers,omitempty"`
	Runs                int       `json:"runs"`
	Tiempos             []float64 `json:"tiempos_seconds"`
	Promedio            float64   `json:"promedio_seconds"`
	MediaRecortada      float64   `json:"media_recortada_seconds"`
	AlertasDetectadas   int       `json:"alertas_detectadas"`
	TiempoLectura       float64   `json:"tiempo_lectura_seconds"`
	TiempoTotalEstimado float64   `json:"tiempo_total_estimado_seconds"`
}

func main() {
	runs       := flag.Int("runs", 20, "Número de ejecuciones de medida")
	numWorkers := flag.Int("workers", 4, "Número de workers")
	outPath    := flag.String("out", "", "Ruta JSON de salida (opcional)")
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

	// ── Fase 1: conteos globales (fuera del benchmark) ───────────────────
	detalleCount := make(map[string]int)
	minuteCount  := make(map[string]int) // clave: "FECHA_HH:MM"
	fantCount    := make(map[string]int)

	for _, row := range data {
		detalle := col(row, idx, "DETALLE_QUEJA")
		hora    := col(row, idx, "HORA_PRESENTACION")
		fecha   := col(row, idx, "FECHA_PRESENTACION_pres")

		if detalle != "" {
			detalleCount[detalle]++
		}
		if len(hora) >= 5 && fecha != "" {
			minuteCount[fecha+"_"+hora[:5]]++
		}
		for _, v := range row {
			v = strings.TrimSpace(v)
			if strings.Contains(v, "EMPRESA_FANTASMA_") {
				fantCount[v]++
			}
		}
	}

	tiempoLectura := time.Since(startLectura).Seconds()

	chunkSize := len(data) / *numWorkers
	if chunkSize == 0 {
		chunkSize = 1
	}

	var tiempos []float64
	alertasFinal := 0

	fmt.Println("====== MODO CONCURRENTE ======")
	fmt.Println("Workers usados:", *numWorkers)

	for i := 1; i <= *runs; i++ {
		start := time.Now()
		alertasFinal = ejecutarConcurrente(data, idx, detalleCount, minuteCount, fantCount, *numWorkers, chunkSize)
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
		Mode:                "concurrente",
		Workers:             *numWorkers,
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

func ejecutarConcurrente(
	data [][]string,
	idx map[string]int,
	detalleCount, minuteCount, fantCount map[string]int,
	numWorkers, chunkSize int,
) int {
	mutex.Lock()
	alertasGlobal = 0
	mutex.Unlock()

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		inicio := i * chunkSize
		fin := inicio + chunkSize
		if i == numWorkers-1 {
			fin = len(data)
		}
		if inicio >= len(data) {
			break
		}
		if fin > len(data) {
			fin = len(data)
		}

		wg.Add(1)
		go worker(data[inicio:fin], idx, detalleCount, minuteCount, fantCount, &wg)
	}

	wg.Wait()
	return alertasGlobal
}

// worker procesa su bloque de forma independiente.
// Acumula en localAlertas sin ningún lock durante el loop.
// Solo adquiere mutex una vez al final para sumar al global.
func worker(
	bloque [][]string,
	idx map[string]int,
	detalleCount, minuteCount, fantCount map[string]int,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	localAlertas := 0

	for _, row := range bloque {
		detalle := col(row, idx, "DETALLE_QUEJA")
		hora    := col(row, idx, "HORA_PRESENTACION")
		fecha   := col(row, idx, "FECHA_PRESENTACION_pres")

		sospechoso := false

		// Patrón 1 — queja duplicada
		if detalleCount[detalle] >= 20000 {
			sospechoso = true
		}

		// Patrón 2 — bombardeo por tiempo:
		// >= 10 registros con misma fecha y mismo HH:MM
		if len(hora) >= 5 && fecha != "" {
			if minuteCount[fecha+"_"+hora[:5]] >= 10 {
				sospechoso = true
			}
		}

		// Patrón 3 — ráfaga nocturna
		if len(hora) >= 2 {
			hh := hora[:2]
			if hh == "00" || hh == "01" || hh == "02" || hh == "03" || hh == "04" {
				sospechoso = true
			}
		}

		// Patrón 4 — denunciado fantasma
		for _, v := range row {
			v = strings.TrimSpace(v)
			if strings.Contains(v, "EMPRESA_FANTASMA_") && fantCount[v] >= 3 {
				sospechoso = true
			}
		}

		if sospechoso {
			localAlertas++
		}
	}

	mutex.Lock()
	alertasGlobal += localAlertas
	mutex.Unlock()
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