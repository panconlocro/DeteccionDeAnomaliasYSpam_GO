# Deteccion de Anomalias y Spam (Go + Python)

Proyecto de analisis de expedientes para detectar patrones sospechosos y comparar desempeno entre procesamiento secuencial y concurrente en Go.

## Objetivo

Este repositorio implementa un flujo completo:

1. Ingesta de archivos fuente (Excel).
2. Merge y estandarizacion inicial.
3. Limpieza de datos.
4. Generacion de dataset sintetico grande.
5. Deteccion de alertas con dos enfoques:
	- secuencial,
	- concurrente.

## Requisitos

- Go 1.21+
- Python 3.10+ (recomendado)

## Estructura principal

- `code/ingestData.py`: descarga archivos fuente desde Google Drive y los separa en `resueltos` y `presentados`.
- `code/mergeData.py`: une y completa datos de expedientes en `data/staging/expedientes_merged.csv`.
- `code/clean_csv/clean_expedientes.go`: limpia y normaliza columnas clave en `data/clean/expedientes_clean.csv`.
- `code/generateData/syntheticData.go`: genera dataset sintetico masivo en `data/synthetic/expedientes_1M.csv`.
- `code/Deteccion_Secuencial/secuencial.go`: deteccion de alertas en una sola goroutine.
- `code/Deteccion_Concurrente/concurrente.go`: deteccion de alertas en paralelo con workers.

## Configuracion de entorno Python

### Windows (PowerShell)

```powershell
./scripts/setup_venv.ps1
./.venv/Scripts/Activate.ps1
```

### macOS/Linux

```bash
bash scripts/setup_venv.sh
source .venv/bin/activate
```

El script instala dependencias de `requirements.txt` y registra el kernel de Jupyter `DeteccionDeAnomalias (venv)`.

## Flujo recomendado de ejecucion

Ejecuta los pasos en este orden desde la raiz del proyecto.

### 1. Ingesta de archivos fuente

```bash
python code/ingestData.py
```

Salida esperada:

- `data/raw/presentados/*`
- `data/raw/resueltos/*`

### 2. Merge de presentados y resueltos

```bash
python code/mergeData.py
```

Salida esperada:

- `data/staging/expedientes_merged.csv`

### 3. Limpieza y normalizacion

```bash
go run ./code/clean_csv/clean_expedientes.go
```

Salida esperada:

- `data/clean/expedientes_clean.csv`

### 4. Generacion de dataset sintetico

Comando por defecto (1M filas):

```bash
go run ./code/generateData/syntheticData.go
```

Comando personalizado:

```bash
go run ./code/generateData/syntheticData.go -in data/clean/expedientes_clean.csv -out data/synthetic/expedientes_1M.csv -n 1000000 -seed 42
```

Salida esperada:

- `data/synthetic/expedientes_1M.csv`

### 5. Deteccion secuencial

```bash
go run ./code/Deteccion_Secuencial/secuencial.go
```

### 6. Deteccion concurrente

```bash
go run ./code/Deteccion_Concurrente/concurrente.go
```

Ambos programas leen el mismo dataset sintetico y reportan:

- alertas detectadas,
- tiempo de lectura,
- promedio de procesamiento,
- media recortada,
- tiempo total estimado.

## Documentacion

- Limpieza de datos: [docs/limpieza.md](docs/limpieza.md)
- Generacion sintetica: [docs/synthetic_data_generation.md](docs/synthetic_data_generation.md)
- Analisis secuencial vs concurrente: [docs/secuencial_vs_concurrente.md](docs/secuencial_vs_concurrente.md)

## Nota sobre datos

Los datos de `data/` pueden ser pesados y no necesariamente se versionan completos en Git. Si te falta informacion para ejecutar el flujo, empieza por `code/ingestData.py`.