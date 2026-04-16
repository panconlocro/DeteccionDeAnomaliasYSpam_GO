# Timestamp Creation (synthts)

Generador concurrente de timestamps sintéticos para el dataset CFPB, preservando la fecha original y agregando señales temporales útiles para detección futura de spam/bots.

## Objetivo

A partir de la columna `Date received`, el programa agrega una nueva columna:

- `synthetic_date_received_ts`

Opcionalmente agrega:

- `synthetic_pattern_type`
- `synthetic_campaign_id`
- `synthetic_is_seeded_suspicious`

La fecha original no se modifica.

## Enfoque de arquitectura

Implementación en 2 pasadas sobre CSV en streaming:

1. Pasada 1 (perfilado global):
- detecta columnas requeridas,
- agrupa por narrativa normalizada + empresa + producto + issue,
- calcula frecuencias para sesgar patrones sospechosos en grupos repetidos.

2. Pasada 2 (enriquecimiento concurrente):
- worker pool configurable (goroutines + channels + WaitGroup),
- generación de timestamp por fila con semilla reproducible,
- escritura ordenada por índice para preservar exactamente el orden del input.

## Patrones sintéticos

No se usa asignación uniforme pura. Se mezcla:

- Comportamiento normal (más diurno/laboral, menos madrugada; weekday/weekend diferenciado).
- Comportamiento sospechoso configurable:
  - burst en ventanas cortas,
  - off-hours (madrugada),
  - intervalos casi regulares,
  - campañas coordinadas.

## Reproducibilidad

- `--seed` define una semilla maestra.
- El RNG se deriva de forma determinística por fila (índice), evitando dependencia del scheduling de goroutines.
- Misma configuración + mismo input => mismo output.

## Requisitos

- Go 1.26.1 (según `go.mod`).
- CSV con estas columnas (se detectan por nombre):
  - `Date received`
  - `Consumer complaint narrative`
  - `Company`
  - `Product`
  - `Issue`
  - `State`

`Date received` soporta:

- `YYYY-MM-DD`
- RFC3339 (si ya viniera con hora)

## Uso

Desde la carpeta `timestamp-creation`:

```bash
go run ./cmd/synthts \
  --input ../data/complaints-2026-04-14_21_03.csv \
  --output ../data/complaints-2026-04-14_21_03.enriched.csv \
  --workers 8 \
  --seed 42 \
  --normal-ratio 0.85 \
  --burst-ratio 0.07 \
  --offhours-ratio 0.04 \
  --coordinated-ratio 0.03 \
  --regular-intervals-ratio 0.01 \
  --burst-window-minutes 10 \
  --add-pattern-columns=true \
  --timezone UTC
```

También puedes correrlo desde la raíz del repo:

```bash
go run ./timestamp-creation/cmd/synthts \
  --input ./data/complaints-2026-04-14_21_03.csv \
  --output ./data/complaints-2026-04-14_21_03.enriched.csv
```

## Flags disponibles

- `--input` (requerido): ruta CSV de entrada.
- `--output` (requerido): ruta CSV de salida.
- `--workers`: cantidad de workers concurrentes (default: 8).
- `--seed`: semilla maestra reproducible (default: 42).
- `--normal-ratio` (default: 0.85).
- `--burst-ratio` (default: 0.07).
- `--offhours-ratio` (default: 0.04).
- `--coordinated-ratio` (default: 0.03).
- `--regular-intervals-ratio` (default: 0.01).
- `--burst-window-minutes` (default: 10).
- `--add-pattern-columns` (default: true).
- `--timezone` (default: UTC).
- `--channel-buffer` (default: 2048).

## Política de errores y consistencia

- No se eliminan filas silenciosamente.
- Si falla el parseo de fecha en una fila:
  - la fila igual se escribe,
  - se incrementa `parse_errors` en el resumen,
  - el patrón puede marcarse como `parse_error_fallback`.
- El orden de salida se preserva igual al input.

## Salida esperada por consola

Al finalizar imprime un resumen, por ejemplo:

```text
rows_processed=1234567 parse_errors=14
pattern_normal=1100000
pattern_burst=70000
pattern_offhours=45000
...
```
