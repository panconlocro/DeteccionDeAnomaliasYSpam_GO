# Detección secuencial vs concurrente en Go

Este documento explica qué hace cada programa, paso por paso, y después resume la diferencia entre ambos enfoques.

## 1. Programa secuencial

Archivo: [code/Deteccion_Secuencial/secuencial.go](../code/Deteccion_Secuencial/secuencial.go)

### Qué hace

1. Define como entrada fija el archivo `data/synthetic/expedientes_1M.csv`.
2. Abre el CSV y lee todo su contenido en memoria con `csv.NewReader(...).ReadAll()`.
3. Elimina la primera fila porque es la cabecera.
4. Mide el tiempo de lectura por separado.
5. Ejecuta 20 veces el mismo proceso para medir rendimiento.
6. En cada ejecución llama a `procesarSecuencial(data)`.
7. `procesarSecuencial` primero recorre todo el dataset para construir un mapa `repetidos` con la cantidad de veces que aparece cada valor de la columna 15, que en el código se llama `detalle`.
8. Luego vuelve a recorrer todos los registros y evalúa si cada fila es sospechosa.
9. Una fila se marca como sospechosa si cumple al menos una de estas reglas:
   - el `detalle` aparece 3 veces o más en el dataset,
   - la hora de la columna 16 empieza con `00:`, `01:`, `02:`, `03:`, `04:` o `05:`,
   - el largo del texto `detalle` es menor que 80 caracteres.
10. Si una fila es sospechosa, incrementa el contador local `alertas`.
11. Al final imprime:
   - alertas detectadas,
   - tiempo de lectura del CSV,
   - promedio de procesamiento,
   - media recortada,
   - tiempo total estimado.

### Qué significa el flujo interno

El programa secuencial hace el trabajo de forma lineal. Primero construye el conteo global de repeticiones y después usa ese conteo para decidir qué filas levantarán alerta. No hay división del trabajo ni sincronización entre hilos porque todo ocurre en una sola goroutine.

## 2. Programa concurrente

Archivo: [code/Deteccion_Concurrente/concurrente.go](../code/Deteccion_Concurrente/concurrente.go)

### Qué hace

1. Usa el mismo archivo de entrada: `data/synthetic/expedientes_1M.csv`.
2. Abre el CSV y lo carga completo en memoria.
3. Elimina la cabecera.
4. Antes de medir el bloque de ejecuciones, construye una vez el mapa `repetidos` con la frecuencia de cada `detalle`.
5. Define `numWorkers := 4` y divide el dataset en 4 bloques del mismo tamaño aproximado.
6. Ejecuta 20 veces `ejecutarConcurrente(data, repetidos, numWorkers, chunkSize)`.
7. `ejecutarConcurrente` reinicia `alertasGlobal` en cero.
8. Luego crea una `sync.WaitGroup` y lanza una goroutine por bloque.
9. Cada goroutine ejecuta `worker(...)` sobre una porción del dataset.
10. Cada worker revisa sus filas con la misma lógica de sospecha que el programa secuencial:
   - frecuencia de `detalle` mayor o igual a 3,
   - hora temprana,
   - `detalle` con menos de 80 caracteres.
11. Cada worker acumula sus resultados en una variable local `localAlertas`.
12. Al terminar, el worker suma su resultado a la variable global `alertasGlobal` dentro de un `sync.Mutex` para evitar condiciones de carrera.
13. `WaitGroup` espera a que terminen todas las goroutines antes de devolver el total.
14. Igual que el secuencial, imprime alertas, tiempo de lectura, promedio, media recortada y tiempo total estimado.

### Qué significa el flujo interno

El programa concurrente divide el trabajo entre varios workers. Cada worker procesa una parte del CSV en paralelo, pero todos leen el mismo mapa `repetidos`, que ya fue calculado antes y no se modifica durante la detección. La sincronización solo se usa para sumar el total final de alertas.

## 3. Diferencia entre secuencial y concurrente

### Secuencial

- Procesa todo en orden, en una sola ejecución lineal.
- No usa goroutines.
- No necesita `Mutex` ni `WaitGroup`.
- Calcula el mapa de repeticiones dentro de cada ejecución de `procesarSecuencial`.
- Es más simple de seguir porque el control de flujo es directo.

### Concurrente

- Divide el dataset en varias partes y las procesa al mismo tiempo.
- Usa 4 goroutines fijas.
- Necesita `WaitGroup` para esperar a todos los workers.
- Necesita `Mutex` para sumar de forma segura el total de alertas.
- Calcula el mapa de repeticiones una sola vez fuera del bucle de benchmark.
- Puede aprovechar mejor varios núcleos si el entorno lo permite.

## 4. Resumen corto

Los dos programas detectan lo mismo con las mismas reglas. La diferencia está en cómo ejecutan el trabajo: el secuencial lo hace paso a paso en una sola rutina, y el concurrente reparte las filas entre varios workers para procesarlas en paralelo.

En este caso, el concurrente no solo gana por paralelismo, sino también porque evita reconstruir el mapa de repeticiones dentro de cada corrida medida.