# Logistic Regression para deteccion de spam

Este modulo reemplaza el enfoque heuristico anterior por un pipeline supervisado en Go para clasificar `ES_SPAM` usando Logistic Regression binaria. El dataset esperado por defecto es:

```bash
data/synthetic/expedientes_synthetic.csv
```

No se usan librerias externas: el loader CSV, TF-IDF, features, entrenamiento, metricas y benchmarks estan implementados con la standard library de Go.

## Ejecutar modo secuencial

Desde la raiz del repositorio:

```bash
go run ./code/Logistic_Rregression/cmd/spamclf \
  -input data/synthetic/expedientes_synthetic.csv \
  -mode sequential \
  -runs 5 \
  -epochs 10 \
  -lr 0.05 \
  -lambda 0.001 \
  -batchSize 1024 \
  -maxFeatures 5000 \
  -minDF 2 \
  -out code/Logistic_Rregression/results/sequential.json
```

## Ejecutar modo concurrente

```bash
go run ./code/Logistic_Rregression/cmd/spamclf \
  -input data/synthetic/expedientes_synthetic.csv \
  -mode concurrent \
  -runs 5 \
  -workers 1,2,4,8,16 \
  -epochs 10 \
  -lr 0.05 \
  -lambda 0.001 \
  -batchSize 1024 \
  -maxFeatures 5000 \
  -minDF 2 \
  -out code/Logistic_Rregression/results/concurrent.json
```

## Comparar secuencial y workers

```bash
go run ./code/Logistic_Rregression/cmd/spamclf \
  -input data/synthetic/expedientes_synthetic.csv \
  -mode compare \
  -runs 5 \
  -workers 1,2,4,8,16 \
  -out code/Logistic_Rregression/results/compare.json
```

Para una prueba rapida sin procesar todo el CSV:

```bash
go run ./code/Logistic_Rregression/cmd/spamclf -mode compare -runs 1 -limit 5000
```

## Features usadas

- Texto: `DETALLE_QUEJA`, normalizado a minusculas, sin tildes basicas, sin puntuacion, tokenizado por espacios.
- TF-IDF: unigramas y bigramas por defecto; configurable con `-bigrams=false`.
- Vocabulario: se ajusta solo con train usando `-maxFeatures` y `-minDF`.
- Timestamp: `TIMESTAMP`; si no existe, intenta `FECHA_PRESENTACION_pres` o `FECHA_PRESENTACION` junto con `HORA_PRESENTACION`.
- Variables temporales: hour, day_of_week, month, is_weekend, is_business_hour, is_night, hour_sin, hour_cos, day_sin y day_cos.
- Categoricas simples: `MATERIA`/`MATERIA_pres` y `TIPO_EXPEDIENTE`/`TIPO_EXPEDIENTE_pres`, con one-hot limitado internamente a los niveles mas frecuentes.

## Features excluidas por leakage

No se usan como input:

- `ES_SPAM`: es la etiqueta.
- `SPAM_SCORE` y `SPAM_TAGS`: fueron generadas junto con la etiqueta y causarian leakage.
- `FECHA_RESOLUCION`, `FORMA_CONCLUSION`, `NRO_RESOLUCION`: informacion posterior o administrativa.
- `NRO_EXPEDIENTE`, `EXPEDIENTE_ORIGEN_pres`, `DOC_DENUNCIADO`, `DENUNCIADOS`: identificadores o entidades crudas que pueden memorizar casos.

## Split y evaluacion

Si todos los registros tienen timestamp parseable, el split es temporal:

- train: registros mas antiguos.
- test: registros mas recientes.

Si no se puede parsear timestamp de forma consistente, usa split aleatorio deterministico con `-seed`.

Metricas reportadas:

- accuracy
- precision
- recall
- f1-score
- matriz de confusion: TP, TN, FP, FN

## Tiempos medidos

Cada run mide:

- lectura CSV
- preprocesamiento/tokenizacion
- construccion de vocabulario y categorias
- vectorizacion TF-IDF/features
- entrenamiento
- evaluacion
- total

El JSON reporta los tiempos por run, promedio y media recortada. La media recortada elimina el menor y mayor tiempo cuando hay mas de dos runs.

## Concurrencia

El modo concurrente usa goroutines, channels y workers en:

- tokenizacion de documentos
- conteo local de document frequency y merge posterior
- vectorizacion TF-IDF
- calculo de gradientes locales por mini-batch y reduce antes de actualizar pesos
- evaluacion/prediccion

Los pesos del modelo no se actualizan desde workers. Cada worker calcula gradientes locales y el hilo principal hace el reduce y la actualizacion, evitando data races.

## Limitaciones

- Logistic Regression es lineal; no captura interacciones complejas como modelos de arboles o redes neuronales.
- La normalizacion de texto es simple y no hace stemming ni lematizacion.
- Las categorias se limitan a niveles frecuentes para mantener la dimensionalidad controlada.
- Los resultados pueden variar levemente entre workers por orden de acumulacion flotante, aunque el pipeline mantiene split, vocabulario, hiperparametros y metricas equivalentes.
