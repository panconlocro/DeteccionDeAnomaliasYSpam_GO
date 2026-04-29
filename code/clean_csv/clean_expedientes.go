package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ── Configuración ────────────────────────────────────────────────────────────

const (
	inputPath  = "expedientes_merged.csv"
	outputPath = "expedientes_clean.csv"
)

// Columnas que se conservan en el output final (en orden).
var keepCols = []string{
	"NRO_EXPEDIENTE",
	"EXPEDIENTE_ORIGEN_pres",
	"TIPO_EXPEDIENTE_pres",
	"FECHA_PRESENTACION_pres",
	"DENUNCIADOS_pres",
	"DOC_DENUNCIADO",
	"MATERIA_pres",
	"año_pres",
	"FECHA_RESOLUCION",
	"NRO_RESOLUCION",
	"FORMA_CONCLUSION",
	"año_res",
	"RES_MATCH_SOURCE",
}

// ── Helpers ──────────────────────────────────────────────────────────────────

var reDateFormats = []string{
	"2006-01-02",
	"02/01/2006",
	"2006/01/02",
}

// parseDate intenta parsear una fecha con varios formatos; devuelve "" si falla.
func parseDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, layout := range reDateFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return "" // fecha inválida → vacía
}

var rePrefixGT = regexp.MustCompile(`^>+\s*`)
var reNewline = regexp.MustCompile(`[\r\n]+`)
var reSpaces = regexp.MustCompile(`\s{2,}`)

// cleanFormaConclusion elimina ">>" iniciales y colapsa saltos de línea.
func cleanFormaConclusion(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Tomar solo la primera "forma" (antes de cualquier \n)
	parts := reNewline.Split(s, -1)
	first := rePrefixGT.ReplaceAllString(strings.TrimSpace(parts[0]), "")
	return strings.TrimSpace(first)
}

// normalizeText colapsa espacios y hace trim.
func normalizeText(s string) string {
	s = strings.TrimSpace(s)
	s = reSpaces.ReplaceAllString(s, " ")
	return s
}

// cleanNroExpediente garantiza formato NNNN-YYYY/TIPO-SUFIJO (trim básico).
func cleanNroExpediente(s string) string {
	return strings.TrimSpace(s)
}

// extractRUC extrae el número de RUC/DNI del campo DOC_DENUNCIADO.
// Ejemplos de entrada: "RUC : 20133840533", "DNI : 12345678"
func extractDocNumber(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return s
}

// validYear devuelve true si el año está en el rango esperado del dataset.
func validYear(s string) bool {
	y, err := strconv.Atoi(strings.TrimSuffix(s, ".0"))
	if err != nil {
		return false
	}
	return y >= 2010 && y <= 2030
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	// Abrir input
	inFile, err := os.Open(inputPath)
	if err != nil {
		log.Fatalf("no se puede abrir %s: %v", inputPath, err)
	}
	defer inFile.Close()

	reader := csv.NewReader(inFile)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Leer encabezado
	header, err := reader.Read()
	if err != nil {
		log.Fatal("error leyendo encabezado:", err)
	}

	// Mapear nombre de columna → índice
	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	// Verificar que las columnas requeridas existen
	for _, col := range keepCols {
		if _, ok := colIdx[col]; !ok {
			log.Fatalf("columna requerida no encontrada: %q", col)
		}
	}

	// Abrir output
	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("no se puede crear %s: %v", outputPath, err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	// Escribir encabezado limpio
	if err := writer.Write(keepCols); err != nil {
		log.Fatal("error escribiendo encabezado:", err)
	}

	// Contadores
	totalRows := 0
	skipped := 0   // filas con NRO_EXPEDIENTE vacío (inválidas)
	noMatch := 0   // expedientes sin resolución
	badDate := 0   // fechas que no pudieron parsearse

	// Procesar filas
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("WARN: error leyendo fila %d: %v", totalRows+1, err)
			continue
		}
		totalRows++

		get := func(col string) string {
			idx, ok := colIdx[col]
			if !ok || idx >= len(row) {
				return ""
			}
			return row[idx]
		}

		// ── 1. Filtrar filas sin NRO_EXPEDIENTE válido ───────────────────
		nroExp := cleanNroExpediente(get("NRO_EXPEDIENTE"))
		if nroExp == "" {
			skipped++
			continue
		}

		// ── 2. Normalizar fechas ─────────────────────────────────────────
		fechaPres := parseDate(get("FECHA_PRESENTACION_pres"))
		fechaRes := parseDate(get("FECHA_RESOLUCION"))
		if get("FECHA_PRESENTACION_pres") != "" && fechaPres == "" {
			badDate++
		}

		// ── 3. Limpiar FORMA_CONCLUSION ──────────────────────────────────
		formaConclusion := cleanFormaConclusion(get("FORMA_CONCLUSION"))

		// ── 4. Normalizar texto libre ────────────────────────────────────
		denunciados := normalizeText(get("DENUNCIADOS_pres"))
		materia := normalizeText(get("MATERIA_pres"))
		tipoExp := normalizeText(get("TIPO_EXPEDIENTE_pres"))
		expOrigen := normalizeText(get("EXPEDIENTE_ORIGEN_pres"))

		// ── 5. Extraer número de documento del denunciado ────────────────
		docDenunciado := extractDocNumber(get("DOC_DENUNCIADO"))

		// ── 6. Validar y limpiar año ─────────────────────────────────────
		añoPres := strings.TrimSuffix(get("año_pres"), ".0")
		if !validYear(añoPres) {
			añoPres = ""
		}
		añoRes := strings.TrimSuffix(get("año_res"), ".0")
		if !validYear(añoRes) {
			añoRes = ""
		}

		// ── 7. Tracking de expedientes sin resolución ────────────────────
		matchSrc := get("RES_MATCH_SOURCE")
		if matchSrc == "none" {
			noMatch++
		}

		// ── 8. Armar fila output en el orden de keepCols ─────────────────
		outRow := []string{
			nroExp,
			expOrigen,
			tipoExp,
			fechaPres,
			denunciados,
			docDenunciado,
			materia,
			añoPres,
			fechaRes,
			get("NRO_RESOLUCION"),
			formaConclusion,
			añoRes,
			matchSrc,
		}

		if err := writer.Write(outRow); err != nil {
			log.Printf("WARN: error escribiendo fila: %v", err)
		}
	}

	writer.Flush()

	// Reporte final
	clean := totalRows - skipped
	fmt.Printf("=== Limpieza completada ===\n")
	fmt.Printf("Filas leídas:              %d\n", totalRows)
	fmt.Printf("Filas sin NRO_EXPEDIENTE:  %d (descartadas)\n", skipped)
	fmt.Printf("Filas en output:           %d\n", clean)
	fmt.Printf("Sin resolución (none):     %d (%.1f%%)\n", noMatch, float64(noMatch)/float64(clean)*100)
	fmt.Printf("Fechas inválidas:          %d\n", badDate)
	fmt.Printf("Output guardado en:        %s\n", outputPath)
}
