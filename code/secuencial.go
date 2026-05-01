package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const inputFile = `C:\Users\MILDRED\Downloads\prograconcu\DeteccionDeAnomaliasYSpam_GO\data\synthetic\expedientes_1M.csv`

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

	tiempoLectura := time.Since(startLectura).Seconds()

	// algoritmo secuencial
	var tiempos []float64
	alertasFinal := 0

	fmt.Println("====== MODO SECUENCIAL ======")

	for i := 1; i <= 20; i++ {

		start := time.Now()

		alertasFinal = procesarSecuencial(data)

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

func procesarSecuencial(data [][]string) int {

	alertas := 0
	repetidos := make(map[string]int)

	// Conteo global
	for _, row := range data {

		if len(row) < 17 {
			continue
		}

		detalle := strings.TrimSpace(row[15])
		repetidos[detalle]++
	}

	// Detección
	for _, row := range data {

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

	copia = copia[1 : len(copia)-1]

	return promedio(copia)
}
