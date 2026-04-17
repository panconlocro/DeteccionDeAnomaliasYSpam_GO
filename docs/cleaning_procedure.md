# Procedimiento de limpieza del dataset

Este documento describe el flujo de limpieza aplicado por el pipeline en Go (carpeta `code/clean_csv`). El objetivo es entregar un CSV limpio con el mismo esquema y un reporte de calidad (QC) para auditoria.

## Objetivo

- Normalizar valores inconsistentes (nulos, espacios, formatos de fecha).
- Estandarizar el texto de las quejas para analisis posterior.
- Mantener el orden y las columnas del CSV original.
- Generar metricas QC de calidad y cobertura.

## Entrada y salida

- Entrada: CSV en `data/` (archivo enriquecido original).
- Salida principal: CSV limpio con las mismas columnas.
- Salida QC: JSON con conteos y estadisticas de limpieza.

## Flujo de limpieza (resumen)

1. Lectura streaming del CSV y normalizacion de header.
2. Limpieza concurrente por fila usando worker pool.
3. Escritura ordenada (se conserva el orden original de filas).
4. Calculo de metricas QC al vuelo.

## Detalle de reglas de limpieza

### 0) Eliminacion de columnas no usadas

Se eliminan de la salida las siguientes columnas, ya que solo se usaban en la data sintetica o son nulas:
- `synthetic_pattern_type`
- `synthetic_campaign_id`
- `synthetic_is_seeded_suspicious`
- `Consumer disputed?`

### 1) Normalizacion de valores nulos

Se convierten a vacio los siguientes tokens (insensible a mayusculas):
- `NaN`, `NA`, `N/A`, `NULL`, `NONE`.

Tambien se hace `trim` de espacios al inicio y fin.

### 2) Normalizacion de IDs numericos

- `Complaint ID` y `ZIP code` se normalizan cuando vienen como numero con `.0`.
- Ejemplo: `12345.0` -> `12345`.

### 3) Normalizacion de fechas

- `Date received` y `Date sent to company` se convierten a formato `YYYY-MM-DD` si se pueden parsear.
- Formatos aceptados: `MM/DD/YY`, `MM/DD/YYYY`, `YYYY-MM-DD`.
- Si la fecha no se puede parsear, se conserva el valor original y se registra en `parse_errors` del QC.

### 4) Normalizacion de timestamp sintetico

- `synthetic_date_received_ts` se convierte a `RFC3339` (UTC) si es parseable.
- Si falla, se conserva el valor original y se registra en `parse_errors`.

### 5) Limpieza del texto narrativo

- Se reemplazan saltos de linea por espacios.
- Se colapsan espacios multiples.
- Se estandarizan redacciones:
  - Cualquier `XX/XX/year>` se normaliza a `XX/XX/YYYY`.
  - Secuencias de `X` de largo >= 2 se normalizan a `XXXX`.

### 6) Filas con columnas faltantes o extra

- Si una fila trae menos columnas, se completa con valores vacios.
- Si trae mas columnas, se trunca al tamano del header.
- Se registran conteos en QC (`rows_short`, `rows_long`).

### 7) Deduplicacion

- Si `-dedup=true`, se elimina duplicado por `Complaint ID`.
- Si falta `Complaint ID`, se usa un hash de respaldo con campos clave:
  `Date received`, `Company`, `Product`, `Issue`, `Consumer complaint narrative`.

## Concurrencia

La limpieza por fila se ejecuta en paralelo con un worker pool configurado por `-workers`.
La escritura conserva el orden original para facilitar auditoria y comparacion.

## Reporte QC (JSON)

El QC incluye:
- Conteos de filas (`rows_in`, `rows_out`, `rows_deduped`).
- Conteo de filas cortas/largas.
- Missingness por columna.
- Errores de parseo por campo.
- Estadisticas de longitud del texto narrativo.
- Conteo de IDs unicos (`unique_complaint_ids`).

## Limitaciones actuales

- No se hace tokenizacion ni limpieza linguistica avanzada.
- No se hacen codificaciones de categorias (solo trim y nulos).
- No se aplica deteccion de idioma ni normalizacion semantica.

## Como ejecutar

```powershell
go run ./code/clean_csv -in data/complaints-2026-04-14_21_03.enriched.csv -out data/complaints-2026-04-14_21_03.cleaned.csv -qc data/complaints-2026-04-14_21_03.qc.json -dedup=true -workers=8
```
