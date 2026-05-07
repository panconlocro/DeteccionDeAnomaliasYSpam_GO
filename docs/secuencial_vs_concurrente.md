# Detección de Spam y Anomalías — Análisis Secuencial vs Concurrente

## 1. Descripción del algoritmo

El sistema detecta patrones de spam en un dataset sintético de 1,000,000 registros del libro de reclamaciones. El algoritmo se divide en dos fases:

**Fase 1 — Conteo global** (ejecutada una sola vez, fuera del benchmark):  
Se recorre el dataset completo para construir dos tablas hash:
- `detalleCount`: frecuencia de cada texto de queja exacto.
- `fantCount`: frecuencia de cada denunciado fantasma (`EMPRESA_FANTASMA_XXXX`).

Esta fase es idéntica en ambas versiones y no se mide en el benchmark porque es un preprocesamiento que en producción se haría una sola vez.

**Fase 2 — Detección** (la que se benchmarkea):  
Se recorre el dataset fila a fila aplicando cuatro reglas sobre los conteos precalculados. Cada fila es independiente de las demás, lo que hace esta fase naturalmente paralelizable.

### Patrones detectados

| Patrón | % en dataset | Regla de detección |
|---|---|---|
| Queja duplicada | 7% | `detalleCount[texto] >= 20000` |
| Bombardeo por tiempo | 8% | `detalleCount[texto] >= 20000` (mismo umbral, textos compartidos) |
| Ráfaga nocturna | 4% | `HORA_PRESENTACION` entre `00:xx` y `04:xx` |
| Denunciado fantasma | 3% | Campo contiene `EMPRESA_FANTASMA_` y `fantCount[v] >= 3` |

> **Nota sobre el umbral `>= 20000`**: los textos de spam provienen de 3 plantillas fijas que se repiten ~23,000 veces cada una. Los textos normales usan plantillas distintas con un máximo de ~16,000 repeticiones. El umbral de 20,000 separa ambas distribuciones sin falsos positivos.

---

## 2. Mecanismos de sincronización

### Versión secuencial

Recorre el dataset fila a fila en un único goroutine. No requiere ningún mecanismo de sincronización. Sirve como línea base para medir el speedup.

### Versión concurrente

Usa tres primitivas de sincronización de Go:

**`sync.WaitGroup`**  
Coordina el ciclo de vida de las goroutines. Se hace `wg.Add(1)` antes de lanzar cada worker y `defer wg.Done()` al inicio de cada uno. El hilo principal llama `wg.Wait()` y bloquea hasta que todos los workers terminen.

**`sync.Mutex`**  
Protege el contador global `alertasGlobal` que es compartido entre todos los workers. Para minimizar la contención, cada worker acumula sus alertas en una variable **local** (`localAlertas`) durante todo su loop, y solo adquiere el mutex **una vez al final** para sumar al global. Esto significa que el lock se adquiere exactamente `numWorkers` veces por ejecución, independientemente del tamaño del dataset.

**Goroutines como workers**  
El dataset se divide en `N` chunks de tamaño `len(data) / numWorkers`. El último worker toma el resto para no perder filas. Cada goroutine procesa su chunk de forma completamente independiente — no comparte estado durante el loop, solo al sumar el resultado final.

```go
// Patrón de uso: localAlertas evita contención en el hot path
func worker(bloque [][]string, ..., wg *sync.WaitGroup) {
    defer wg.Done()
    localAlertas := 0          // sin lock
    for _, row := range bloque {
        if esSospechoso(row) {
            localAlertas++     // sin lock
        }
    }
    mutex.Lock()               // lock solo una vez
    alertasGlobal += localAlertas
    mutex.Unlock()
}
```

---

## 3. Speedup y media recortada

El benchmark corre 20 ejecuciones de la Fase 2 para cada versión y calcula:

**Media recortada**: se ordenan los tiempos, se elimina el mínimo y el máximo, y se promedia el resto. Esto reduce el impacto de outliers causados por el scheduler del SO o picos de CPU de otros procesos.

**Speedup**:

$$S = \frac{T_{secuencial}}{T_{concurrente}}$$

Donde ambos tiempos son la media recortada de las 20 ejecuciones.

**Eficiencia paralela**:

$$E = \frac{S}{N_{workers}}$$

Un valor de E cercano a 1.0 indica que los workers están siendo aprovechados óptimamente. Valores bajos indican overhead de sincronización o cuellos de botella en memoria.

---

## 4. Análisis de speedup, scalability y trade-offs

### Speedup esperado

Con 4 workers sobre una tarea CPU-bound y paralelizable, la ley de Amdahl predice un speedup teórico máximo cercano a 4x. En la práctica se espera menos debido a:

- Overhead de creación de goroutines (mínimo en Go, pero existe).
- Contención de memoria: todos los workers leen los mismos mapas `detalleCount` y `fantCount` desde caché. Con datasets grandes esto puede causar cache misses.
- El scheduler de Go (GOMAXPROCS) puede no mapear goroutines 1:1 a núcleos físicos dependiendo de la máquina.

### Scalability

La versión concurrente escala bien en la Fase 2 porque:

- No hay escrituras compartidas durante el procesamiento (solo lecturas de los mapas precalculados).
- La única escritura compartida (`alertasGlobal`) ocurre una vez por worker, no una vez por fila.
- El trabajo está distribuido uniformemente (chunks de igual tamaño).

El límite de escalabilidad está en la Fase 1 (conteo global), que permanece secuencial. Si el dataset creciera 10x, la Fase 1 dominaría el tiempo total y el speedup de la Fase 2 se volvería irrelevante.

### Trade-offs

| Aspecto | Secuencial | Concurrente |
|---|---|---|
| Complejidad del código | Baja | Media |
| Riesgo de bugs | Bajo | Mayor (race conditions si se usa mal el mutex) |
| Tiempo de procesamiento | Mayor | Menor |
| Uso de CPU | 1 núcleo | N núcleos |
| Uso de memoria | Sin overhead | Mínimo overhead por goroutines (~2KB c/u) |
| Reproducibilidad | Determinista | Determinista (alertas), no determinista (tiempos) |

---

## 5. Uso y rendimiento de recursos

### CPU

La versión concurrente distribuye la Fase 2 entre N workers que corren en paralelo en núcleos distintos (Go usa `GOMAXPROCS = núcleos físicos` por defecto). El uso de CPU esperado es proporcional al número de workers hasta el límite de núcleos disponibles.

### Memoria

Ambas versiones cargan el dataset completo en RAM (`reader.ReadAll()`). Para 1,000,000 filas con ~20 columnas, el uso aproximado es:

- Dataset en memoria: ~800 MB – 1.2 GB dependiendo del largo de los strings.
- Mapas de conteo (`detalleCount`, `fantCount`): ~50–100 MB adicionales.
- Overhead de goroutines: ~2 KB por goroutine × 4 workers = despreciable.

### Observación sobre I/O

La lectura del CSV (Fase 1 de I/O) es el cuello de botella más grande en tiempo absoluto. Paralelizar la Fase 2 reduce el tiempo de procesamiento pero no el tiempo de lectura, por lo que el speedup total (incluyendo I/O) es menor que el speedup de procesamiento puro.