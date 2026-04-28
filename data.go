package main

import (
	"encoding/csv"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ==========================================================
// CONFIG
// ==========================================================
const inputFolder = `C:\Users\MILDRED\Downloads\uploads`
const outputFile = `C:\Users\MILDRED\Downloads\expedientes_1M_FINAL.csv`

const targetRows = 1000000

// ==========================================================
// CATÁLOGOS
// ==========================================================
var comentariosUsuario = []string{
	"No me solucionan nada desde hace meses y sigo esperando una respuesta clara.",
	"El banco me hizo un cobro que no reconozco y nadie me brinda una explicación.",
	"Ya llamé varias veces al servicio al cliente y no me dan solución.",
	"Me están cobrando algo que no corresponde según mi contrato.",
	"Nunca me notificaron correctamente sobre este procedimiento.",
	"Esto me parece un abuso hacia los consumidores.",
	"Me siento estafado por la forma en que manejaron mi caso.",
	"No respetaron lo que decía el contrato firmado inicialmente.",
	"El servicio recibido fue pésimo y la atención deficiente.",
	"No es justo lo que están haciendo conmigo como cliente.",
	"He presentado reclamos anteriores y hasta ahora no obtengo respuesta.",
	"La empresa no cumple con lo ofrecido al momento de venderme el servicio.",
	"Estoy siendo perjudicado económicamente por errores de la entidad.",
	"Nadie se hace responsable por el problema que me generaron.",
	"Solicito una solución inmediata porque llevo meses esperando.",
	"Cada vez que llamo me transfieren y nadie resuelve nada.",
	"Me indicaron una cosa al inicio y luego cambiaron las condiciones.",
	"Considero que se están vulnerando mis derechos como consumidor.",
	"La atención fue deficiente y no mostraron interés en ayudarme.",
	"El problema persiste pese a que ya presenté varios reclamos.",
	"Se comprometieron a devolverme el dinero y nunca cumplieron.",
	"No recibí información clara ni transparente sobre los cargos aplicados.",
	"El producto entregado no coincide con lo ofrecido en la publicidad.",
	"Me cobraron penalidades sin previo aviso ni sustento.",
	"Estoy cansado de insistir y no obtener ninguna respuesta concreta.",
	"La plataforma nunca funcionó correctamente y aun así me cobraron.",
	"Solicité la cancelación del servicio y continúan facturándome.",
	"No respetaron los plazos establecidos para resolver mi reclamo.",
	"El personal fue descortés y no quiso registrar mi queja.",
	"Me prometieron una solución en 48 horas y nunca ocurrió.",
}

var comentariosSpam = []string{
	"ayuda ayuda ayuda ayuda",
	"reclamo reclamo reclamo",
	"no responden no responden",
	"estafa estafa estafa",
	"urgente urgente urgente",
	"quiero solución ya ya ya",
	"nadie responde nadie responde",
	"fraude fraude fraude",
	"devuelvan dinero devuelvan dinero",
	"mal servicio mal servicio mal servicio",
}

var detalles = []string{
	"Expediente relacionado con reclamo de consumo.",
	"Caso en evaluación por la comisión competente.",
	"Revisión administrativa en curso.",
	"Expediente derivado a segunda instancia.",
	"Solicitud de atención registrada correctamente.",
}

// ==========================================================
// GLOBAL
// ==========================================================
var dataReal [][]string
var header []string
var mu sync.Mutex

// ==========================================================
// HELPERS
// ==========================================================
func randChoice(arr []string) string {
	return arr[rand.Intn(len(arr))]
}

func limpiarTexto(txt string) string {
	txt = strings.ReplaceAll(txt, "\n", " | ")
	txt = strings.ReplaceAll(txt, "\r", " ")
	txt = strings.TrimSpace(txt)
	return txt
}

func horaNormal() string {
	h := rand.Intn(10) + 8
	return fmt.Sprintf("%02d:%02d:%02d", h, rand.Intn(60), rand.Intn(60))
}

func horaBot() string {
	h := rand.Intn(7)
	return fmt.Sprintf("%02d:%02d:%02d", h, rand.Intn(60), rand.Intn(60))
}

func generarComentario() (string, string, int) {

	flag := 0
	hora := horaNormal()

	if rand.Float64() < 0.08 {
		flag = 1
		hora = horaBot()
		return randChoice(comentariosSpam), hora, flag
	}

	txt := randChoice(comentariosUsuario)

	if rand.Float64() < 0.10 {
		txt = strings.ToUpper(txt)
	}

	if rand.Float64() < 0.15 {
		txt += "!!!"
	}

	return txt, hora, flag
}

// ==========================================================
// LEER CSV
// ==========================================================
func leerArchivo(path string, wg *sync.WaitGroup) {
	defer wg.Done()

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil || len(rows) == 0 {
		return
	}

	mu.Lock()

	if len(header) == 0 {
		header = rows[0]
	}

	for i := 1; i < len(rows); i++ {

		row := rows[i]

		for j := range row {
			row[j] = limpiarTexto(row[j])
		}

		dataReal = append(dataReal, row)
	}

	mu.Unlock()

	fmt.Println("Leído:", filepath.Base(path), "Filas:", len(rows)-1)
}

// ==========================================================
// MAIN
// ==========================================================
func main() {

	rand.Seed(time.Now().UnixNano())

	fmt.Println("Buscando archivos CSV...")

	var archivos []string

	filepath.WalkDir(inputFolder, func(path string, d fs.DirEntry, err error) error {

		if strings.HasSuffix(strings.ToLower(path), ".csv") {
			archivos = append(archivos, path)
		}

		return nil
	})

	if len(archivos) == 0 {
		fmt.Println("No se encontraron CSV.")
		return
	}

	// ======================================================
	// LEER REALES
	// ======================================================
	var wg sync.WaitGroup

	for _, archivo := range archivos {
		wg.Add(1)
		go leerArchivo(archivo, &wg)
	}

	wg.Wait()

	fmt.Println("Registros reales:", len(dataReal))

	// ======================================================
	// OUTPUT
	// ======================================================
	out, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	writer := csv.NewWriter(out)
	defer writer.Flush()

	nuevoHeader := append(header,
		"COMENTARIO_USUARIO",
		"DETALLE_EXPEDIENTE",
		"HORA_PRESENTACION",
		"FLAG_SOSPECHOSO",
	)

	writer.Write(nuevoHeader)

	total := 0

	// ======================================================
	// ESCRIBIR REALES
	// ======================================================
	for _, row := range dataReal {

		com, hora, flag := generarComentario()

		nueva := append(row,
			com,
			randChoice(detalles),
			hora,
			fmt.Sprintf("%d", flag),
		)

		writer.Write(nueva)
		total++
	}

	// ======================================================
	// COMPLETAR HASTA 1M
	// ======================================================
	fmt.Println("Generando filas sintéticas faltantes...")

	for total < targetRows {

		base := dataReal[rand.Intn(len(dataReal))]

		clon := make([]string, len(base))
		copy(clon, base)

		// pequeñas mutaciones
		if len(clon) > 0 {
			clon[0] = fmt.Sprintf("%s-X%d", clon[0], total)
		}

		com, hora, flag := generarComentario()

		nueva := append(clon,
			com,
			randChoice(detalles),
			hora,
			fmt.Sprintf("%d", flag),
		)

		writer.Write(nueva)

		total++

		if total%100000 == 0 {
			fmt.Println(total, "filas...")
		}
	}

	fmt.Println("===================================")
	fmt.Println("Archivo generado correctamente")
	fmt.Println("Ruta:", outputFile)
	fmt.Println("Total filas:", total)
	fmt.Println("===================================")
}
