# Informe Técnico — Detección de Anomalías y Spam (Go + Promela/SPIN)

**Repositorio:** `panconlocro/DeteccionDeAnomaliasYSpam_GO`  
**Fecha:** 2026-05-10  
**Analista:** Arquitecto de Software Senior (análisis automático sobre código fuente real)  
**Versión analizada:** commit `df0c48c` — versión final enviada por el autor

---

## 1. Calidad de código

### 1.1 Organización del proyecto

La estructura de directorios es clara y sigue las convenciones idiomáticas de Go:

```
code/
├── Logistic_Rregression/      ← clasificador principal
│   ├── cmd/spamclf/main.go
│   └── internal/{benchmark,data,features,model,pipeline,text}/
├── generateData/              ← generador de datos sintéticos
│   ├── cmd/main.go
│   └── internal/{anomly,generator,model,sampler,textgen,timeline,validator,writer}/
├── clean_csv/                 ← limpieza del dataset crudo
├── promela/                   ← modelo formal
└── notebooks/                 ← exploración Python
```

La separación `cmd/` (punto de entrada) + `internal/` (lógica de dominio) es una buena práctica de Go que impide importaciones cruzadas no deseadas. El módulo de clasificación exhibe la granularidad correcta: cada responsabilidad tiene su propio paquete (`data`, `features`, `model`, `pipeline`, `text`, `benchmark`).

### 1.2 Modularidad y legibilidad

| Componente | Evaluación |
|---|---|
| `pipeline/common.go` | Excelente: orquesta el pipeline completo (secuencial/concurrente) con bifurcación limpia por `concurrent bool` |
| `pipeline/config.go` | Bueno: struct de configuración con método `normalize()` defensivo |
| `model/logistic.go` | Muy bueno: implementación de regresión logística con mini-batch y gradiente concurrente, clara y bien estructurada |
| `features/timestamp.go` | Muy bueno: 10 features temporales (cíclicas con seno/coseno) bien documentadas implícitamente en la constante `TimestampFeatureCount = 10` |
| `text/tokenizer.go` | Bueno: normalización eficiente usando `strings.Builder` y `strings.NewReplacer` (optimizado por Go) |
| `generateData/cmd/main.go` | Regular: toda la orquestación en `main()` con configuración hardcodeada, sin flags CLI |


### 1.3 Buenas prácticas en Go

**Positivo:**
- Uso de `sync.WaitGroup` y canales con cierre explícito (`close(jobs)`) para señalizar fin de trabajo.
- `defer file.Close()` presente en todos los aperturas de archivo del clasificador y `clean_csv`.
- Manejo de errores con retorno explícito en todas las funciones del clasificador.
- `csv.NewReader` con `FieldsPerRecord = -1` y `LazyQuotes = true` para tolerancia a datos reales sucios.
- `rand.New(rand.NewSource(seed))` en lugar del RNG global — reproducibilidad garantizada.
- `append([]Record(nil), ...)` en `split.go` para copiar slices sin aliasing.

**A mejorar:**
- `min` y `max` redefinidos en `model/minmax.go` cuando Go 1.21 (versión usada según `go.mod`) ya los incluye como builtins.
- `removeAccents` en `features/categorical.go` duplica `accentReplacer` de `text/tokenizer.go` y `clean_csv/clean_expedientes.go`. Son tres implementaciones separadas de la misma lógica, ligeramente inconsistentes entre sí (la de `categorical.go` no cubre mayúsculas con tilde).
- Typos en nombres de carpetas y paquetes: `Logistic_Rregression` (doble `r`), `anomly` (falta la `a`).
- `generateData/cmd/main.go`: rutas de entrada/salida y `totalRows` hardcodeadas; no hay flags CLI.
- `NewCSVWriter` en `generateData/internal/writer/csv_writer.go` tiene una fuga de descriptor de archivo: si `w.Write(header)` falla, retorna error pero el archivo `f` ya fue abierto y no se cierra.
- `w.Writer.Flush()` en el `main` de `generateData` se llama sin verificar errores de escritura a disco.
- `GenerationContext.mu` (campo `sync.Mutex`) está declarado pero nunca se usa — el loop principal de generación es completamente secuencial.

### 1.4 Resumen de calidad de código

| Criterio | Puntuación |
|---|---|
| Organización | ★★★★★ |
| Modularidad (clasificador) | ★★★★★ |
| Modularidad (generateData) | ★★★☆☆ |
| Legibilidad | ★★★★☆ |
| Buenas prácticas Go | ★★★★☆ |
| Manejo de errores | ★★★☆☆ |
| Nomenclatura | ★★★☆☆ |
| Código duplicado | ★★★☆☆ |

---

## 2. Concurrencia

### 2.1 Patrón global

El pipeline tiene **5 etapas concurrentizables**, cada una con su propio patrón de paralelismo. El orquestador en `pipeline/common.go` bifurca correctamente entre la versión secuencial y concurrente de cada etapa.

### 2.2 Análisis por etapa

#### Etapa 1 — Preprocesamiento / Tokenización
**Patrón:** Worker pool con canal de índices (`jobs chan int`) + `WaitGroup`.

```go
// pipeline/common.go
jobs := make(chan int)
var wg sync.WaitGroup
for worker := 0; worker < workers; worker++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for i := range jobs {
            record := records[i]
            record.Tokens = txt.Tokenize(record.Text, useBigrams)
            out[i] = record  // escritura en posición pre-asignada
        }
    }()
}
for i := range records { jobs <- i }
close(jobs)
wg.Wait()
```

**Evaluación:** Correcto. Cada worker escribe en `out[i]` donde `i` es único — no hay contención. `close(jobs)` señaliza fin de trabajo correctamente. No hay races.

#### Etapa 2 — Construcción del vocabulario (Document Frequency)
**Patrón:** Chunk-based. Cada worker calcula un `map[string]int` local sobre su chunk, y el goroutine principal hace la fusión.

```go
// pipeline/common.go — buildVectorizerConcurrent
results := make(chan map[string]int, actualWorkers)
// ... workers envían su DF local al canal ...
wg.Wait()
close(results)
merged := make(map[string]int)
for local := range results { for token, count := range local { merged[token] += count } }
```

**Evaluación:** Correcto y eficiente. El canal bufferizado con capacidad `actualWorkers` evita bloqueos. La fusión es serial pero O(V × W) donde V es el vocabulario y W los workers — despreciable frente al cómputo de DF.

#### Etapa 3 — Encoder de categorías
**Patrón:** Mismo patrón chunk + fusión. Cada worker mantiene mapas locales `materias` y `tipos`.

**Evaluación:** Correcto. Sin contención.

#### Etapa 4 — Vectorización TF-IDF
**Patrón:** Worker pool idéntico al preprocesamiento. Cada worker lee `vectorizer` (solo lectura — seguro) y escribe `out[i]` en su índice único.

**Evaluación:** Correcto. No hay races porque `Vectorizer` es inmutable tras su construcción.

#### Etapa 5 — Entrenamiento (mini-batch gradiente paralelo)
**Patrón:** Por cada mini-batch, se divide el batch en `actualWorkers` chunks. Cada worker acumula gradientes localmente en `grads[worker]` (array pre-asignado por worker). El hilo principal reduce y aplica el gradiente.

```go
// model/logistic.go — TrainConcurrent
grads := make([][]float64, actualWorkers)
biasGrads := make([]float64, actualWorkers)
// workers escriben en grads[worker] (su propio slice)
wg.Wait()
// reducción serial
grad := make([]float64, dimension)
for worker := 0; worker < actualWorkers; worker++ {
    for j := 0; j < dimension; j++ { grad[j] += grads[worker][j] }
}
m.applyGradient(grad, biasGrad, batchSize, cfg)
```

**Evaluación:** Correcto. Los workers nunca escriben en `m.Weights` — cada uno tiene su propio `grads[worker]`. La actualización de pesos es serial y segura. Esta es la implementación correcta del patrón **Hogwild-safe** con reducción explícita.

#### Etapa 6 — Evaluación
**Patrón:** Workers con métricas locales (`local Metrics`) + `sync.Mutex` para fusionar en `out Metrics` global.

```go
// model/metrics.go — EvaluateConcurrent
var mu sync.Mutex
// ...
mu.Lock()
out.TP += local.TP; out.TN += local.TN; out.FP += local.FP; out.FN += local.FN
mu.Unlock()
```

**Evaluación:** Correcto. El uso de `sync.Mutex` aquí es apropiado porque la sección crítica es pequeña (4 sumas de enteros) y los workers realizan la mayor parte del trabajo (predicción) fuera del lock.

### 2.3 Análisis de riesgos

| Riesgo | Estado | Detalle |
|---|---|---|
| Race conditions | ✅ No detectadas | Escrituras en posiciones únicas o con lock |
| Deadlocks | ✅ No detectados | Canales con `close()` correcto; `WaitGroup` bien balanceado |
| Goroutine leaks | ✅ No detectados | Todos los workers terminan al cerrar el canal `jobs` o al agotar chunks |
| Starvation | ✅ No aplica | El scheduler de Go es preemptivo |
| `GenerationContext.mu` sin usar | ⚠️ GAP | Mutex declarado pero el loop de generación es monohilo |

### 2.4 Escalabilidad y cuellos de botella

- La reducción de gradientes por mini-batch es **O(D × W)** (D = dimensión, W = workers). Para D = 5000 features y W = 16 workers, esto introduce trabajo serial no despreciable que limita el speedup del entrenamiento.
- La etapa de `training` escala peor que `vectorization` y `preprocess` porque la fase serial (reducción + actualización de pesos) crece con más workers sin aportar más paralelismo.
- Correctamente mitigado: el sistema reporta el mejor número de workers por `TrimmedMean`, no asume que más siempre es mejor.

---

## 3. Rendimiento

### 3.1 Resultados reales (benchmark ejecutado con dataset de 969.115 filas)

| Configuración | Tiempo promedio sin CSV (s) | Speedup vs Secuencial |
|---|---|---|
| Secuencial | 14.97 | 1.00× |
| Concurrente 1 worker | 17.50 | 0.86× (overhead canales) |
| Concurrente 2 workers | 11.80 | 1.27× |
| Concurrente 4 workers | ~8.00 | ~1.87× |
| Concurrente 6 workers | ~6.28 | ~2.38× |
| Concurrente 8 workers | 6.16 | 2.43× |
| **Concurrente 10 workers (mejor)** | **6.09** | **2.46×** |
| Concurrente 16 workers | 6.13 | 2.44× |

> Los resultados de 4-7 workers se interpolaron de las trazas parciales del archivo `compare_runs.txt`. Los valores de 8, 10 y 16 workers son mediciones directas del benchmark.

### 3.2 Desglose por etapa (comparativa secuencial vs 10 workers)

| Etapa | Secuencial (s) | 10 workers (s) | Speedup etapa |
|---|---|---|---|
| `read_csv` | 0.607 | 0.590 | 1.03× (I/O, esperado) |
| `preprocess` | 2.075 | 0.868 | **2.39×** |
| `vocabulary` | 3.672 | 1.033 | **3.55×** |
| `vectorization` | 6.747 | 2.219 | **3.04×** |
| `training` | 1.329 | 0.876 | **1.52×** |
| `evaluation` | 0.022 | 0.005 | **4.40×** |

### 3.3 Observaciones de rendimiento

**Overhead de workers=1 concurrente:**  
Con 1 worker, la versión concurrente tarda **17.50s vs 14.97s** de la secuencial — un overhead del **~17%**. Esto se debe al costo de crear goroutines, el scheduler de canales y la asignación de slices auxiliares (`grads[]`). Es un comportamiento esperado y correcto que el sistema detecta y reporta.

**Plateau de escalabilidad a ~10 workers:**  
El speedup se estabiliza en ~2.46× más allá de 10 workers. Esto es consistente con la Ley de Amdahl aplicada a la fracción serial del pipeline. Las etapas `training` y `vocabulary` tienen componentes seriales significativas (reducción de gradientes y fusión de mapas) que limitan el speedup global.

**Etapa dominante original (vectorización):**  
La vectorización TF-IDF es la etapa más lenta en el modo secuencial (6.75s, 45% del tiempo total de cómputo). Escala muy bien (3.04×) porque es embarazosamente paralela: cada ejemplo se vectoriza de forma independiente.

**Uso de memoria:**  
El dataset completo (~969K filas) se carga en memoria como `[]Record`. Para el dataset actual es viable, pero para datasets >5 millones de filas podría requerir streaming. El uso de `SparseVector` ([]Feature con índice+valor) en lugar de vectores densos de 5000 floats es una decisión correcta que reduce el uso de memoria entre 10× y 100× según la densidad del texto.

### 3.4 Metodología de medición

- **Trimmed mean:** se descarta el mejor y el peor tiempo de 5 runs. Es la métrica más robusta para comparar tiempos de ejecución. ✅
- **Separación CSV vs cómputo:** el tiempo de lectura del CSV se excluye del speedup principal porque es I/O y no escala con workers. Esto es metodológicamente correcto. ✅
- **Determinismo:** las métricas de clasificación (TP, TN, FP, FN) son idénticas entre la versión secuencial y concurrente para todos los números de workers — confirma que la paralelización no introduce errores numéricos. ✅

---

## 4. Seguridad y robustez

### 4.1 Validación de entradas

| Validación | Implementación | Evaluación |
|---|---|---|
| Columnas requeridas en CSV | `buildIndex()` + checks explícitos en `LoadRecords` | ✅ Correcto |
| Labels binarios | `parseLabel()` con múltiples variantes ("spam", "si", "1", "true", etc.) | ✅ Robusto |
| Parámetros de configuración | `Config.normalize()` con defaults defensivos | ✅ Correcto |
| Workers válidos | `validateWorkers()` y `clampWorkers()` | ✅ Correcto |
| Acceso a índices de features | `if f.Index >= 0 && f.Index < dimension` en `PredictProba` | ✅ Correcto |
| Threshold del modelo | Validado en `NewLogisticRegression` y `normalizeConfig` | ✅ Correcto |
| Filas vacías / dataset vacío | Detectado en `LoadRecords` | ✅ Correcto |
| Ruta de output | `os.MkdirAll` antes de crear archivo | ✅ Correcto |

### 4.2 Manejo de archivos y recursos

**Fuga de descriptor en `NewCSVWriter`:**
```go
// generateData/internal/writer/csv_writer.go
f, err := os.Create(path)
// ...
err = w.Write(header)
if err != nil {
    return nil, err  // ← f queda abierto sin cerrarse
}
```
Si la escritura del header falla (disco lleno, permisos), el archivo `f` queda abierto. Corrección: agregar `f.Close()` antes de retornar el error.

**`Flush()` sin chequeo de error:**
```go
// generateData/cmd/main.go
w.Writer.Flush()
// ← debería ser: if err := w.Writer.Flush(); err != nil { log.Fatalf(...) }
```
Si el buffer no se puede escribir a disco (disco lleno), el error silencioso produce un CSV truncado sin aviso.

**Manejo correcto en el clasificador:**  
`clean_csv/clean_expedientes.go` usa `defer writer.Flush()` y `defer outFile.Close()` apropiadamente. ✅

### 4.3 Rutas hardcodeadas

```go
// generateData/cmd/main.go
inputCSV := "data/clean/expedientes_clean.csv"
outputCSV := "data/synthetic/expedientes_synthetic.csv"
totalRows := 1_000_000
```

No hay flags CLI en `generateData`. Para ejecutar con diferentes rutas o configuraciones hay que recompilar. Esto no es una vulnerabilidad de seguridad, pero sí reduce la robustez operativa.

### 4.4 Robustez frente a datasets grandes

- La lectura completa en memoria (`records := make([]Record, 0)` + append) es razonable para el dataset actual (~969K filas ≈ ~1-2 GB de vectores dispersos).
- El generador de datos sintéticos recorta `RecentComplaints` al superar 50.000 entradas (control de memoria correcto).
- No hay límite de memoria configurado para la carga del CSV en el clasificador — un dataset de 10M+ filas podría causar OOM.

---

## 5. Verificación formal (Promela/SPIN)

Archivo analizado: `code/promela/ModeladoInicial.pml`.

### 5.1 Descripción del modelo

```promela
#define N_WORKERS 4
#define N_REGISTROS 5

int alertas_globales = 0;
bool mutex = true;
int data[N_WORKERS * N_REGISTROS];

proctype Worker(byte id) {
    // Acumula alertas locales en su chunk
    atomic {
        (mutex == true) -> mutex = false;
        alertas_globales = alertas_globales + alertas_locales;
        mutex = true;
    }
    workers_done++
}
```

El modelo captura el patrón central: N workers procesan chunks de datos independientemente, acumulan resultados locales y los fusionan en un contador global protegido por un mutex simulado con `bool`.

### 5.2 Evaluación del modelo Promela

| Criterio | Estado | Detalle |
|---|---|---|
| Representación de concurrencia | ✅ Correcto | N workers independientes procesando en paralelo |
| Sección crítica | ✅ Correcto | `atomic { (mutex == true) -> ... }` es una técnica válida en Promela |
| Fusión de resultados | ✅ Correcto | Suma serial tras release del mutex |
| `liveness` LTL | ✅ Correcto | `<> (workers_done == N_WORKERS)` — eventualmente todos terminan |
| `safety` LTL | ✅ Correcto pero trivial | `[] (alertas_globales >= 0)` — siempre no negativo (entero siempre ≥ 0 en Promela) |
| Relación con código Go | ⚠️ Parcial | Capta el patrón de acumulación, no el pipeline completo |
| Propiedades funcionales | ❌ Ausente | No hay propiedad que verifique que `alertas_globales == suma_esperada` |
| Cobertura del pipeline | ❌ Limitada | Solo representa la etapa de "evaluación/conteo", no preprocess, vocabulary ni training |

### 5.3 Análisis de la propiedad de safety

La propiedad `ltl safety { [] (alertas_globales >= 0) }` es semánticamente correcta pero trivialmente verdadera en Promela, donde los enteros son no-negativos por definición de la asignación `alertas_globales = alertas_globales + alertas_locales` (ambos valores son siempre ≥ 0). Una propiedad más útil sería:

```promela
// Verificar que la suma es correcta al finalizar
ltl correctitud { <> (workers_done == N_WORKERS && alertas_globales == 10) }
// donde 10 es la suma conocida del array data[] inicializado
```

### 5.4 Limitaciones del modelo formal

1. **Cobertura de etapas:** El modelo solo cubre el patrón de acumulación/conteo (similar a la evaluación y DF). No modela preprocesamiento, tokenización, vectorización ni entrenamiento.
2. **Escala:** N_WORKERS=4, N_REGISTROS=5 — escala de juguete vs 969K registros reales.
3. **Sin evidencia de ejecución SPIN:** No hay trazas de verificación (`pan.c`, trail files) ni scripts de automatización en el repositorio.
4. **Mutex como bool:** Idiomático en Promela, pero difiere del `sync.Mutex` de Go. Una representación más fiel usaría un canal bufferizado de capacidad 1.
5. **Propiedad de safety trivial:** No aporta garantías funcionales reales.

---

## 6. GAPs identificados

| Dimensión | GAP | Severidad |
|---|---|---|
| **Nomenclatura** | Typos: `Logistic_Rregression` (doble r), `anomly` (falta a) | Baja |
| **Código duplicado** | `removeAccents`/`accentReplacer` definidos 3 veces (tokenizer, categorical, clean_csv) con inconsistencias entre sí (la de categorical no cubre mayúsculas con tilde) | Media |
| **Builtins** | `min`/`max` redefinidos en `model/minmax.go` cuando Go 1.21 los provee como builtins | Baja |
| **Robustez I/O** | Fuga de descriptor en `NewCSVWriter` si falla escritura del header | Media |
| **Robustez I/O** | `w.Writer.Flush()` sin chequeo de error en `generateData/cmd/main.go` | Media |
| **Diseño** | `GenerationContext.mu` declarado pero nunca adquirido — el generador es single-threaded | Baja |
| **Configuración** | Rutas y `totalRows` hardcodeadas en `generateData` — sin flags CLI | Media |
| **Concurrencia** | El generador de datos sintéticos (`generateData`) es completamente secuencial — genera 1M de filas sin usar goroutines | Baja-Media |
| **Testing** | Tests unitarios solo para el módulo clasificador; `generateData`, `clean_csv` y el pipeline integrado no tienen tests | Alta |
| **Testing** | No hay tests de `go test -race` (detector de race conditions de Go) | Media |
| **Escalabilidad** | Carga completa del dataset en RAM; no hay modo streaming para datasets >5M filas | Media |
| **Verificación formal** | Modelo Promela no cubre el pipeline completo (vocabulary, vectorization, training) | Media |
| **Verificación formal** | Propiedad LTL de safety trivialmente verdadera; no hay propiedad de correctitud funcional | Media |
| **Verificación formal** | No hay evidencia de ejecución de SPIN en el repositorio | Media |
| **Modelo** | `generateData/cmd/main.go` tiene toda la lógica de orquestación en `main()` | Baja |

---

## 7. Recomendaciones

### 7.1 Alta prioridad

**R1 — Corregir fuga de descriptor en `NewCSVWriter`:**
```go
// Antes
err = w.Write(header)
if err != nil {
    return nil, err
}
// Después
err = w.Write(header)
if err != nil {
    f.Close()
    return nil, err
}
```

**R2 — Verificar error de `Flush()` en `generateData`:**
```go
// Antes
w.Writer.Flush()
// Después
if err := w.Writer.Flush(); err != nil {
    log.Fatalf("error al flush final: %v", err)
}
```

**R3 — Agregar tests de integración para `generateData` y `clean_csv`:**  
La ausencia de tests en estos módulos es el gap más crítico. Al menos un test de tabla pequeña que valide las transformaciones de `normalizeDenunciado`, `cleanFormaConclusion`, y la generación de un subconjunto de filas sintéticas.

### 7.2 Media prioridad

**R4 — Centralizar la normalización de texto:**  
Crear un paquete compartido (p.ej., `internal/normalize`) con una única función `RemoveAccents(string) string` que cubra correctamente mayúsculas y minúsculas, usada desde `text/tokenizer.go`, `features/categorical.go` y `clean_csv`.

**R5 — Agregar flags CLI a `generateData`:**
```go
inputCSV  := flag.String("input",     "data/clean/expedientes_clean.csv", "CSV de entrada")
outputCSV := flag.String("output",    "data/synthetic/expedientes_synthetic.csv", "CSV de salida")
totalRows := flag.Int("rows",         1_000_000, "filas a generar")
seed      := flag.Int64("seed",       0, "seed (0 = basado en tiempo)")
flag.Parse()
```

**R6 — Eliminar `model/minmax.go` y usar builtins de Go 1.21:**
```go
// Eliminar model/minmax.go
// En model/logistic.go, min() y max() de Go 1.21 funcionan directamente
end := min(start+cfg.BatchSize, len(examples))
```

**R7 — Corregir o eliminar `GenerationContext.mu`:**  
Si el generador se mantiene single-threaded, remover el campo `mu` para no inducir a error. Si se planea paralelizar la generación, implementar el uso del mutex.

**R8 — Corrección de typos en nombres de paquetes/carpetas:**  
`Logistic_Rregression` → `LogisticRegression` o `logistic_regression`  
`anomly` → `anomaly`

### 7.3 Baja prioridad / mejoras opcionales

**R9 — Fortalecer las propiedades LTL del modelo Promela:**
```promela
// Calcular la suma esperada y verificarla
#define SUMA_ESPERADA 10
ltl correctitud { <> (workers_done == N_WORKERS && alertas_globales == SUMA_ESPERADA) }
```

**R10 — Documentar y ejecutar SPIN:**  
Agregar un script `scripts/run_spin.sh` con la invocación de SPIN y un archivo de trazas de verificación. Esto hace el trabajo formal reproducible.

**R11 — Ampliar el modelo Promela:**  
Agregar etapas de `vocabulary_build` (con merge de mapas) y `training` (con reducción de gradientes) para que el modelo refleje mejor la arquitectura real del sistema.

**R12 — Considerar streaming para el clasificador:**  
Para datasets >5M filas, implementar lectura lazy del CSV que procese bloques en lugar de cargar todo en memoria.

---

## 8. Conclusión

### Evaluación general

El proyecto implementa un sistema concurrente de detección de spam bien concebido y correctamente ejecutado en su núcleo. Los puntos fuertes son claros:

**Fortalezas principales:**
- **Concurrencia correcta y verificable:** los 6 patrones concurrentes implementados (worker pool por canal, chunk+merge, gradiente paralelo con reducción serial, evaluación con mutex) son todos correctos. No se detectaron race conditions ni deadlocks en el análisis estático ni hay inconsistencias en los resultados de benchmark (las métricas son idénticas entre versiones secuencial y concurrente).
- **Speedup real y medido:** 2.46× de aceleración con 10 workers sobre 969K registros reales, con metodología de medición rigurosa (trimmed mean, separación de I/O del cómputo, múltiples runs).
- **Pipeline bien estructurado:** la separación secuencial/concurrente en `pipeline/common.go` es limpia, testeable y extensible.
- **Calidad del clasificador:** regresión logística con TF-IDF + bigramas + features temporales cíclicas + encoding de categorías es un modelo sólido para el dominio del problema.
- **Infraestructura de benchmark:** el módulo `benchmark` con `Report`, `CompareReport`, `TrimmedMean` y salida JSON+TXT es de calidad profesional.

**Áreas de mejora:**
- Hay un gap de testing significativo fuera del módulo clasificador.
- La generación de datos sintéticos tiene deuda técnica menor (rutas hardcodeadas, fuga de descriptor, mutex sin usar).
- El modelo Promela es funcional pero simplificado frente a la complejidad real del pipeline.
- La duplicación de la lógica de normalización de texto en tres lugares distintos es el único GAP de diseño notable en el código limpio.

### Evaluación académica

Desde la perspectiva de un curso de **Programación Concurrente y Distribuida**, el proyecto demuestra comprensión sólida de los mecanismos de sincronización de Go (`goroutines`, `sync.WaitGroup`, `sync.Mutex`, canales), implementa patrones de concurrencia apropiados para el problema (worker pools, map-reduce local), y produce resultados de performance medibles y justificables. La elección del modelo de verificación formal (Promela/SPIN) con propiedades LTL es coherente con el dominio del curso.

**Calificación técnica global: 8.5/10** — Proyecto sólido con implementación concurrente correcta y funcional, con oportunidades de mejora en robustez operativa, cobertura de tests y profundidad del modelo formal.

