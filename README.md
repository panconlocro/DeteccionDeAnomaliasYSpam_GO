# Deteccion de Anomalias y Spam en Expedientes (Go + Python)

Pipeline completo para:

1. Ingerir y unificar expedientes reales (INDECOPI).
2. Limpiar y normalizar datos administrativos.
3. Generar un dataset sintetico con spam/anomalias inyectadas.
4. Entrenar y benchmarkear un clasificador de spam con Logistic Regression en Go.
5. Comparar ejecucion secuencial vs concurrente con metricas y tiempos por etapa.

El proyecto esta pensado para trabajo reproducible en cursos de analisis de datos, sistemas concurrentes y ML aplicado.

## Stack

- Go 1.21
- Python 3.10+ (scripts de ingesta/merge y notebooks)
- Sin librerias externas en el pipeline Go de modelado (solo standard library)

## Estructura del proyecto

- [code/ingestData.py](code/ingestData.py): descarga carpeta de Google Drive y separa archivos en `presentados` y `resueltos`.
- [code/mergeData.py](code/mergeData.py): merge de excels historicos y export a staging.
- [code/clean_csv/clean_expedientes.go](code/clean_csv/clean_expedientes.go): limpieza fuerte de columnas clave.
- [code/generateData/cmd/main.go](code/generateData/cmd/main.go): generador sintetico (1,000,000 filas por defecto).
- [code/Logistic_Rregression/cmd/spamclf/main.go](code/Logistic_Rregression/cmd/spamclf/main.go): CLI principal para entrenamiento/evaluacion/benchmark.
- [scripts/setup_venv.ps1](scripts/setup_venv.ps1), [scripts/setup_venv.sh](scripts/setup_venv.sh): setup de entorno Python + kernel Jupyter.

## Flujo end-to-end

Ejecuta estos pasos desde la raiz del repositorio.

### 1) Configurar entorno Python

Windows (PowerShell):

```powershell
./scripts/setup_venv.ps1
./.venv/Scripts/Activate.ps1
```

macOS/Linux:

```bash
bash scripts/setup_venv.sh
source .venv/bin/activate
```

### 2) Ingesta de datos fuente

```bash
python code/ingestData.py
```

Genera carpetas con excels en:

- `data/raw/presentados/`
- `data/raw/resueltos/`

### 3) Merge de datasets historicos

```bash
python code/mergeData.py
```

Salida:

- `data/staging/expedientes_merged.csv`

### 4) Limpieza y normalizacion

```bash
go run ./code/clean_csv/clean_expedientes.go
```

Salida:

- `data/clean/expedientes_clean.csv`

### 5) Generacion sintetica

```bash
go run ./code/generateData/cmd/main.go
```

Salida:

- `data/synthetic/expedientes_synthetic.csv`

Nota: el generador actual usa rutas internas fijas (`data/clean/...` -> `data/synthetic/...`) y genera 1,000,000 filas por defecto.

### 6) Entrenamiento y benchmarking (Logistic Regression)

#### Secuencial

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

#### Concurrente (multi-worker)

```bash
go run ./code/Logistic_Rregression/cmd/spamclf \
	-input data/synthetic/expedientes_synthetic.csv \
	-mode concurrent \
	-workers 1,2,4,8,16 \
	-runs 5 \
	-out code/Logistic_Rregression/results/concurrent.json
```

#### Comparativo secuencial vs concurrente

```bash
go run ./code/Logistic_Rregression/cmd/spamclf \
	-input data/synthetic/expedientes_synthetic.csv \
	-mode compare \
	-workers 1,2,4,8,16 \
	-runs 5 \
	-out code/Logistic_Rregression/results/compare.json
```

Smoke test rapido:

```bash
go run ./code/Logistic_Rregression/cmd/spamclf -mode compare -runs 1 -limit 5000
```

## Salidas y artefactos

Cuando usas `-out` en el CLI de spam, se generan:

- Un JSON con metricas y tiempos.
- Un TXT `*_runs.txt` con detalle por corrida.

Ejemplo:

- `code/Logistic_Rregression/results/compare.json`
- `code/Logistic_Rregression/results/compare_runs.txt`

En las metricas, el proyecto reporta accuracy, precision, recall, F1 y matriz de confusion (TP/TN/FP/FN), ademas de tiempos por stage (read_csv, preprocess, vocabulary, vectorization, training, evaluation, total).

## Dataset y features (resumen)

- Label: `ES_SPAM`.
- Texto principal: `DETALLE_QUEJA` con TF-IDF (unigramas y bigramas opcionales).
- Features temporales derivadas de `TIMESTAMP` (o fallback a fecha/hora de presentacion).
- Variables categoricas limitadas para controlar dimensionalidad.

Columnas excluidas por leakage (ejemplo): `SPAM_SCORE`, `SPAM_TAGS`, `ES_SPAM` como feature, e identificadores administrativos crudos.

## Calidad y reproducibilidad

- Split temporal cuando hay timestamps consistentes; si no, split aleatorio deterministico por `seed`.
- Modo concurrente paralelo por workers, sin actualizar pesos desde goroutines (reduce en hilo principal).
- Para comparar performance de procesamiento puro, el reporte separa tiempos sin lectura CSV y tiempos totales con lectura incluida.

## Documentacion adicional

- [docs/merge_strategy.md](docs/merge_strategy.md)
- [docs/limpieza.md](docs/limpieza.md)
- [docs/synthetic_data_generation.md](docs/synthetic_data_generation.md)
- [docs/deteccionRegressionLigistica.md](docs/deteccionRegressionLigistica.md)
- [docs/secuencial_vs_concurrente.md](docs/secuencial_vs_concurrente.md)

## Troubleshooting rapido

- Error de activacion en PowerShell: ejecuta `Set-ExecutionPolicy -Scope Process -ExecutionPolicy RemoteSigned`.
- Si faltan datos en `data/raw/`, vuelve a correr [code/ingestData.py](code/ingestData.py).
- Si quieres limpiar artefactos y volver a correr todo, elimina archivos en `data/staging/`, `data/clean/`, `data/synthetic/` y `code/Logistic_Rregression/results/`.

## Licencia

Este repositorio incluye licencia MIT en [LICENSE](LICENSE).