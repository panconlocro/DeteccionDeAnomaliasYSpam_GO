package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Parametros de generacion y distribucion de patrones.
const (
	tipoNormal  = 0
	tipoBomb    = 1
	tipoDup     = 2
	tipoNoct    = 3
	tipoFant    = 4
	bombGroup   = 10
	dupGroup    = 8
	fantGroup   = 5
	pctBomb     = 0.08
	pctDup      = 0.07
	pctNoct     = 0.04
	pctFant     = 0.03
	progressMod = 100000
)

// Plantillas y componentes de texto sintetico.
var (
	spamTexts = []string{
		"Presento queja formal por incumplimiento de contrato y solicito sancion inmediata.",
		"Solicito sancion urgente contra la empresa por danos al consumidor.",
		"Queja por incumplimiento. Solicito resolucion inmediata del caso.",
	}
	acciones = []string{
		"Presento una",
		"Formulo una",
		"Interpongo una",
		"Registro una",
	}
	motivos = []string{
		"por",
		"debido a",
		"a causa de",
	}
	solicitudes = []string{
		"Solicito investigacion y medidas.",
		"Pido una respuesta formal y pronta.",
		"Solicito sancion si corresponde.",
		"Requiero solucion y seguimiento.",
	}
	detallesExtra = []string{
		"No hubo respuesta oportuna.",
		"Se incumplieron condiciones acordadas.",
		"El servicio recibido fue insatisfactorio.",
	}
)

func main() {
	inPath := flag.String("in", "data/clean/expedientes_clean.csv", "Input CSV")
	outPath := flag.String("out", "data/synthetic/expedientes_1M.csv", "Output CSV")
	target := flag.Int("n", 1_000_000, "Target rows")
	seed := flag.Int64("seed", 42, "Random seed")
	flag.Parse()

	header, rows, err := readCSV(*inPath)
	if err != nil {
		log.Fatalf("read input: %v", err)
	}
	if len(rows) == 0 {
		log.Fatal("input has no rows")
	}

	outHeader, inToOut, outIndex := buildHeader(header)

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		log.Fatalf("mkdir output: %v", err)
	}

	outFile, err := os.Create(*outPath)
	if err != nil {
		log.Fatalf("create output: %v", err)
	}
	defer outFile.Close()

	buf := bufio.NewWriterSize(outFile, 1024*1024)
	writer := csv.NewWriter(buf)
	if err := writer.Write(outHeader); err != nil {
		log.Fatalf("write header: %v", err)
	}

	rng := rand.New(rand.NewSource(*seed))
	// Precalcular etiquetas spam/normal para respetar la distribucion objetivo.
	tipos := buildTipos(*target, rng)

	// Estado para patrones de spam por grupos.
	bombRemain := 0
	bombHour := 0
	bombMinute := 0
	dupRemain := 0
	dupText := ""
	fantRemain := 0
	fantDen := ""

	for i := 0; i < *target; i++ {
		// Muestreo con reemplazo desde las filas base.
		base := rows[rng.Intn(len(rows))]
		out := make([]string, len(outHeader))
		for inIdx, outIdx := range inToOut {
			if inIdx >= 0 && inIdx < len(base) {
				out[outIdx] = base[inIdx]
			}
		}

		materia := normalizeToken(getField(out, outIndex, "MATERIA_pres"), "servicio")
		tipo := normalizeToken(getField(out, outIndex, "TIPO_EXPEDIENTE_pres"), "queja")
		denunciado := normalizeToken(getField(out, outIndex, "DENUNCIADOS_pres"), "la empresa")

		if idx, ok := outIndex["FECHA_PRESENTACION_pres"]; ok {
			out[idx] = variarFecha(out[idx], rng)
		}
		if idx, ok := outIndex["NRO_EXPEDIENTE"]; ok {
			out[idx] = variarExpediente(out[idx], rng)
		}

		detalle := detalleNormal(materia, tipo, denunciado, rng)
		hora := horaNormal(rng)
		esSpam := "0"
		tipoSpam := "normal"

		// Aplicar el patron de spam seleccionado (o mantener normal).
		switch tipos[i] {
		case tipoBomb:
			if bombRemain == 0 {
				bombRemain = bombGroup
				bombHour = 8 + rng.Intn(10)
				bombMinute = rng.Intn(60)
			}
			bombRemain--
			hora = fmt.Sprintf("%02d:%02d:%02d", bombHour, bombMinute, rng.Intn(60))
			detalle = spamTexts[rng.Intn(len(spamTexts))]
			esSpam = "1"
			tipoSpam = "bombardeo_tiempo"
		case tipoDup:
			if dupRemain == 0 {
				dupRemain = dupGroup
				dupText = spamTexts[rng.Intn(len(spamTexts))]
			}
			dupRemain--
			detalle = dupText
			esSpam = "1"
			tipoSpam = "queja_duplicada"
		case tipoNoct:
			hora = fmt.Sprintf("%02d:%02d:%02d", rng.Intn(5), rng.Intn(60), rng.Intn(60))
			esSpam = "1"
			tipoSpam = "rafaga_nocturna"
		case tipoFant:
			if fantRemain == 0 {
				fantRemain = fantGroup
				fantDen = fmt.Sprintf("EMPRESA_FANTASMA_%04d", rng.Intn(10000))
			}
			fantRemain--
			setField(out, outIndex, "DENUNCIADOS_pres", fantDen)
			hora = fmt.Sprintf("%02d:%02d:%02d", 8+rng.Intn(10), rng.Intn(60), rng.Intn(60))
			esSpam = "1"
			tipoSpam = "denunciado_fantasma"
		}

		setField(out, outIndex, "HORA_PRESENTACION", hora)
		setField(out, outIndex, "DETALLE_QUEJA", detalle)
		setField(out, outIndex, "ES_SPAM", esSpam)
		setField(out, outIndex, "TIPO_SPAM", tipoSpam)

		if err := writer.Write(out); err != nil {
			log.Fatalf("write row %d: %v", i, err)
		}
		if (i+1)%progressMod == 0 {
			fmt.Printf("%d/%d filas generadas...\n", i+1, *target)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Fatalf("flush: %v", err)
	}
	if err := buf.Flush(); err != nil {
		log.Fatalf("buffer flush: %v", err)
	}
	fmt.Printf("Listo: %d filas en %s\n", *target, *outPath)
}

func readCSV(path string) ([]string, [][]string, error) {
	// Lee el CSV completo en memoria para permitir muestreo aleatorio.
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return nil, nil, err
	}

	var rows [][]string
	for {
		rec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, err
		}
		rows = append(rows, rec)
	}

	return header, rows, nil
}

func buildHeader(header []string) ([]string, []int, map[string]int) {
	// Asegura columnas requeridas y construye indices de acceso rapido.
	outHeader := append([]string{}, header...)
	outIndex := make(map[string]int, len(outHeader))
	for i, name := range outHeader {
		outIndex[name] = i
	}

	required := []string{"ES_SPAM", "TIPO_SPAM", "DETALLE_QUEJA", "HORA_PRESENTACION"}
	for _, col := range required {
		if _, ok := outIndex[col]; !ok {
			outIndex[col] = len(outHeader)
			outHeader = append(outHeader, col)
		}
	}

	inToOut := make([]int, len(header))
	for i, name := range header {
		inToOut[i] = outIndex[name]
	}
	return outHeader, inToOut, outIndex
}

func buildTipos(total int, rng *rand.Rand) []uint8 {
	tipos := make([]uint8, total)
	// Calcula conteos por porcentaje y luego mezcla el orden.
	nBomb := int(float64(total) * pctBomb)
	nDup := int(float64(total) * pctDup)
	nNoct := int(float64(total) * pctNoct)
	nFant := int(float64(total) * pctFant)
	nNormal := total - (nBomb + nDup + nNoct + nFant)

	idx := 0
	for i := 0; i < nNormal; i++ {
		tipos[idx] = tipoNormal
		idx++
	}
	for i := 0; i < nBomb; i++ {
		tipos[idx] = tipoBomb
		idx++
	}
	for i := 0; i < nDup; i++ {
		tipos[idx] = tipoDup
		idx++
	}
	for i := 0; i < nNoct; i++ {
		tipos[idx] = tipoNoct
		idx++
	}
	for i := 0; i < nFant; i++ {
		tipos[idx] = tipoFant
		idx++
	}

	rng.Shuffle(len(tipos), func(i, j int) {
		tipos[i], tipos[j] = tipos[j], tipos[i]
	})
	return tipos
}

func getField(row []string, index map[string]int, name string) string {
	// Devuelve el valor de una columna por nombre si existe.
	idx, ok := index[name]
	if !ok || idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func setField(row []string, index map[string]int, name, value string) {
	// Asigna el valor de una columna por nombre si existe.
	idx, ok := index[name]
	if !ok || idx < 0 || idx >= len(row) {
		return
	}
	row[idx] = value
}

func normalizeToken(value, fallback string) string {
	// Normaliza valores vacios o "nan" a un fallback.
	v := strings.TrimSpace(value)
	if v == "" {
		return fallback
	}
	if strings.EqualFold(v, "nan") {
		return fallback
	}
	return v
}

func detalleNormal(materia, tipo, denunciado string, rng *rand.Rand) string {
	// Genera un detalle normal con plantillas y variacion ligera.
	tipo = strings.ToLower(tipo)
	materia = strings.ToLower(materia)
	if materia == "" {
		materia = "servicio"
	}
	if denunciado == "" {
		denunciado = "la empresa"
	}

	s1 := fmt.Sprintf("%s %s contra %s %s %s.",
		acciones[rng.Intn(len(acciones))],
		tipo,
		denunciado,
		motivos[rng.Intn(len(motivos))],
		materia,
	)
	s2 := solicitudes[rng.Intn(len(solicitudes))]
	if rng.Float64() < 0.35 {
		s3 := detallesExtra[rng.Intn(len(detallesExtra))]
		return strings.TrimSpace(s1 + " " + s2 + " " + s3)
	}
	return strings.TrimSpace(s1 + " " + s2)
}

func horaNormal(rng *rand.Rand) string {
	// Genera hora con distribucion realista de oficina.
	hour := 0
	if rng.Float64() < 0.85 {
		hour = 8 + rng.Intn(10)
	} else {
		hour = 18 + rng.Intn(4)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hour, rng.Intn(60), rng.Intn(60))
}

func variarFecha(fecha string, rng *rand.Rand) string {
	// Aplica un deslizamiento de dias para diversificar fechas.
	fecha = strings.TrimSpace(fecha)
	if fecha == "" || strings.EqualFold(fecha, "nan") {
		return fecha
	}
	t, err := time.Parse("2006-01-02", fecha)
	if err != nil {
		return fecha
	}
	delta := rng.Intn(31) - 15
	return t.AddDate(0, 0, delta).Format("2006-01-02")
}

func variarExpediente(nro string, rng *rand.Rand) string {
	// Genera o reemplaza el sufijo numerico del expediente.
	n := strings.TrimSpace(nro)
	if n == "" {
		return fmt.Sprintf("EXP-%06d", rng.Intn(900000)+100000)
	}
	idx := strings.LastIndex(n, "-")
	if idx == -1 {
		return fmt.Sprintf("%s-%06d", n, rng.Intn(900000)+100000)
	}
	prefix := n[:idx]
	return fmt.Sprintf("%s-%06d", prefix, rng.Intn(900000)+100000)
}
