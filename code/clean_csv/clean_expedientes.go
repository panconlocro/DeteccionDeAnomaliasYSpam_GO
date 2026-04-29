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
    inputPath  = "data/staging/expedientes_merged.csv"
    outputPath = "data/clean/expedientes_clean.csv"
)

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

// accentReplacer elimina diacríticos (tildes, ñ, ü).
// Problema de origen: Excel 2017 tenía tildes, 2018-2021 no.
var accentReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
	"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
	"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
	"À", "A", "È", "E", "Ì", "I", "Ò", "O", "Ù", "U",
	"ñ", "n", "Ñ", "N",
	"ü", "u", "Ü", "U",
)

// reLegalDot elimina los puntos dentro de formas legales tipo S.A. → SA,
// S.A.A. → SAA, S.A.C. → SAC, E.I.R.L. → EIRL, LTDA. → LTDA, INC. → INC
// Solo actúa sobre puntos entre letras mayúsculas o al final de una secuencia
// de letras mayúsculas, para no tocar puntos de abreviatura en nombres propios.
var reLegalDot = regexp.MustCompile(`(?:[A-Z])\.`)

// reNOrdinal normaliza variantes del ordinal N°: "N° 1", "N°1", "N°  2", "Nº 1" → "N1", "N2"
var reNOrdinal = regexp.MustCompile(`N[°º]?\s*\.?\s*(\d)`)

// reSpaces colapsa múltiples espacios.
var reSpaces = regexp.MustCompile(`\s{2,}`)

var rePrefixGT = regexp.MustCompile(`^>+\s*`)
var reNewline = regexp.MustCompile(`[\r\n]+`)

// normalizeDenunciado unifica variantes ortográficas del mismo nombre de entidad.
//
// Tres problemas de origen que resuelve:
//  1. Tildes: Excel 2017 las tenía, 2018-2021 no  →  eliminamos todas
//  2. Puntos en forma legal: S.A.A. / SAA / S.A.A  →  eliminamos puntos internos
//  3. Ordinal de comisión: N° 1 / N°1 / N°  1 / 1  →  normalizamos a N1, N2...
//
// Resultado: 136 grupos de duplicados (~3,646 filas, ~26% del campo) quedan unificados.
// Forma canónica de salida: sin tildes, sin puntos en razón social, sin espacios extra.
// Ejemplo: "BANCO DE CRÉDITO DEL PERÚ S.A." → "BANCO DE CREDITO DEL PERU SA"
//          "COMISIÓN DE PROTECCIÓN AL CONSUMIDOR N° 1" → "COMISION DE PROTECCION AL CONSUMIDOR N1"
//          "SCOTIABANK PERÚ S.A.A." → "SCOTIABANK PERU SAA"
func normalizeDenunciado(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 1. Tildes
	s = accentReplacer.Replace(s)
	// 2. Mayúsculas (el dataset ya está en mayúsculas, pero por si acaso)
	s = strings.ToUpper(s)
	// 3. Ordinal N° → N (antes de tocar puntos para no confundir)
	s = reNOrdinal.ReplaceAllString(s, "N$1")
	// 4. Puntos en forma legal (S.A. → SA)
	s = reLegalDot.ReplaceAllStringFunc(s, func(m string) string {
		return string(m[0]) // quitar el punto, conservar la letra
	})
	// 5. Colapsar espacios
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
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
	return ""
}

// cleanFormaConclusion elimina ">>" iniciales y se queda con la primera forma
// cuando hay múltiples concatenadas con \n (ej. ">>CONFIRMA\n>>CONSENTIDO").
func cleanFormaConclusion(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := reNewline.Split(s, -1)
	first := rePrefixGT.ReplaceAllString(strings.TrimSpace(parts[0]), "")
	return strings.TrimSpace(first)
}

// normalizeText colapsa espacios múltiples y hace trim.
func normalizeText(s string) string {
	s = strings.TrimSpace(s)
	return reSpaces.ReplaceAllString(s, " ")
}

// extractDocNumber extrae el número de RUC/DNI del campo DOC_DENUNCIADO.
// Entrada: "RUC : 20133840533" → Salida: "20133840533"
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
	inFile, err := os.Open(inputPath)
	if err != nil {
		log.Fatalf("no se puede abrir %s: %v", inputPath, err)
	}
	defer inFile.Close()

	reader := csv.NewReader(inFile)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		log.Fatal("error leyendo encabezado:", err)
	}

	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	for _, col := range keepCols {
		if _, ok := colIdx[col]; !ok {
			log.Fatalf("columna requerida no encontrada: %q", col)
		}
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("no se puede crear %s: %v", outputPath, err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	if err := writer.Write(keepCols); err != nil {
		log.Fatal("error escribiendo encabezado:", err)
	}

	totalRows := 0
	skipped := 0
	noMatch := 0
	badDate := 0

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

		// 1. Filtrar filas sin NRO_EXPEDIENTE válido
		nroExp := strings.TrimSpace(get("NRO_EXPEDIENTE"))
		if nroExp == "" {
			skipped++
			continue
		}

		// 2. Normalizar fechas a YYYY-MM-DD
		fechaPres := parseDate(get("FECHA_PRESENTACION_pres"))
		fechaRes := parseDate(get("FECHA_RESOLUCION"))
		if get("FECHA_PRESENTACION_pres") != "" && fechaPres == "" {
			badDate++
		}

		// 3. Limpiar FORMA_CONCLUSION (quitar ">>" y quedarse con la primera)
		formaConclusion := cleanFormaConclusion(get("FORMA_CONCLUSION"))

		// 4. Normalizar texto libre (espacios + trim)
		tipoExp := normalizeText(get("TIPO_EXPEDIENTE_pres"))
		expOrigen := normalizeText(get("EXPEDIENTE_ORIGEN_pres"))
		materia := normalizeText(get("MATERIA_pres"))

		// 5. Normalizar denunciados: tildes + puntos de razón social + ordinal N°
		denunciados := normalizeDenunciado(get("DENUNCIADOS_pres"))

		// 6. Extraer número de documento (quitar prefijo "RUC :" / "DNI :")
		docDenunciado := extractDocNumber(get("DOC_DENUNCIADO"))

		// 7. Limpiar años (pandas los serializa como float: "2017.0" → "2017")
		añoPres := strings.TrimSuffix(get("año_pres"), ".0")
		if !validYear(añoPres) {
			añoPres = ""
		}
		añoRes := strings.TrimSuffix(get("año_res"), ".0")
		if !validYear(añoRes) {
			añoRes = ""
		}

		matchSrc := get("RES_MATCH_SOURCE")
		if matchSrc == "none" {
			noMatch++
		}

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

	clean := totalRows - skipped
	fmt.Printf("=== Limpieza completada ===\n")
	fmt.Printf("Filas leídas:              %d\n", totalRows)
	fmt.Printf("Filas sin NRO_EXPEDIENTE:  %d (descartadas)\n", skipped)
	fmt.Printf("Filas en output:           %d\n", clean)
	fmt.Printf("Sin resolución (none):     %d (%.1f%%)\n", noMatch, float64(noMatch)/float64(clean)*100)
	fmt.Printf("Fechas inválidas:          %d\n", badDate)
	fmt.Printf("Output guardado en:        %s\n", outputPath)
}
