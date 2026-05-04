package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const inputFile = "data/synthetic/expedientes_1M.csv"

var mutex sync.Mutex
var alertasGlobal int

func main() {

	// lectura del archivo
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

	data = data[1:] // quitar cabecera

	repetidos := make(map[string]int)
	for _, row := range data {
		if len(row) < 17 {
			continue
		}
		detalle := strings.TrimSpace(row[15])
		repetidos[detalle]++
	}

	tiempoLectura := time.Since(startLectura).Seconds()

	numWorkers := 4
	chunkSize := len(data) / numWorkers

	var tiempos []float64
	alertasFinal := 0

	fmt.Println("====== MODO CONCURRENTE ======")
	fmt.Println("Workers usados:", numWorkers)

	for i := 1; i <= 20; i++ {

		start := time.Now()

		alertasFinal = ejecutarConcurrente(data, repetidos, numWorkers, chunkSize)

		elapsed := time.Since(start).Seconds()
		tiempos = append(tiempos, elapsed)

		fmt.Printf("Ejecución %d: %.6f segundos\n", i, elapsed)
	}

	prom := promedio(tiempos)
	rec := mediaRecortada(tiempos)

	fmt.Println("--------------------------------")
	fmt.Println("Alertas detectadas:", alertasFinal)
	fmt.Printf("Tiempo lectura CSV: %.6f s\n", tiempoLectura)
	fmt.Printf("Promedio procesamiento: %.6f s\n", prom)
	fmt.Printf("Media recortada: %.6f s\n", rec)
	fmt.Printf("Tiempo total estimado: %.6f s\n", tiempoLectura+rec)
}

func ejecutarConcurrente(data [][]string, repetidos map[string]int, numWorkers, chunkSize int) int {

	alertasGlobal = 0

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {

		inicio := i * chunkSize
		fin := inicio + chunkSize

		if i == numWorkers-1 {
			fin = len(data)
		}

		wg.Add(1)
		go worker(data[inicio:fin], repetidos, &wg)
	}

	wg.Wait()

	return alertasGlobal
}

func worker(bloque [][]string, repetidos map[string]int, wg *sync.WaitGroup) {
	defer wg.Done()

	localAlertas := 0

	for _, row := range bloque {

		if len(row) < 17 {
			continue
		}

		detalle := strings.TrimSpace(row[15])
		hora := strings.TrimSpace(row[16])

		sospechoso := false

		if repetidos[detalle] >= 3 {
			sospechoso = true
		}

		if strings.HasPrefix(hora, "00:") ||
			strings.HasPrefix(hora, "01:") ||
			strings.HasPrefix(hora, "02:") ||
			strings.HasPrefix(hora, "03:") ||
			strings.HasPrefix(hora, "04:") ||
			strings.HasPrefix(hora, "05:") {
			sospechoso = true
		}

		if len(detalle) < 80 {
			sospechoso = true
		}

		if sospechoso {
			localAlertas++
		}
	}

	mutex.Lock()
	alertasGlobal += localAlertas
	mutex.Unlock()
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

	copia = copia[1 : len(copia)-1]

	return promedio(copia)
}
