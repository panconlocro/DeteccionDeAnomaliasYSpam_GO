# README — Limpieza de Datos: Expedientes INDECOPI

## Contexto

El dataset `expedientes_merged.csv` es el resultado de combinar dos fuentes de datos administrativos del INDECOPI (2017–2021): expedientes **presentados** ante la Sala Especializada en Protección al Consumidor (SPC) y expedientes **resueltos** por la misma. El merge fue realizado en Python mediante dos estrategias: por `NRO_EXPEDIENTE` y por `EXPEDIENTE_ORIGEN`.

Como resultado del merge, el CSV tiene columnas duplicadas con sufijos `_pres` y `_res`, campos de texto con formatos inconsistentes heredados de los Excel originales, y columnas auxiliares del merge que no tienen utilidad para el análisis posterior.

El script `clean_expedientes.go` se encarga de dejarlo en un estado usable.

---

## Qué hace el script, paso a paso

### 1. Selección de columnas

El merge generó 18 columnas, varias de ellas duplicadas (versión `_pres` y versión `_res` del mismo dato). El script descarta las columnas redundantes de la fuente `_res` (ya que el dato canónico es el del presentado) y conserva solo las 13 columnas con información relevante:

| Columna conservada       | Descripción                                      |
|--------------------------|--------------------------------------------------|
| `NRO_EXPEDIENTE`         | Identificador único del expediente en la SPC     |
| `EXPEDIENTE_ORIGEN_pres` | Expediente origen en la comisión regional        |
| `TIPO_EXPEDIENTE_pres`   | Tipo: APELACION, QUEJA, MEDIDA CAUTELAR, etc.    |
| `FECHA_PRESENTACION_pres`| Fecha en que ingresó a la SPC                    |
| `DENUNCIADOS_pres`       | Nombre del denunciado                            |
| `DOC_DENUNCIADO`         | RUC o DNI del denunciado (limpiado)              |
| `MATERIA_pres`           | Materia del caso (ej. TARJETA DE CREDITO)        |
| `año_pres`               | Año de presentación                              |
| `FECHA_RESOLUCION`       | Fecha en que se resolvió el expediente           |
| `NRO_RESOLUCION`         | Número de la resolución emitida                  |
| `FORMA_CONCLUSION`       | Resultado: CONFIRMA, REVOCA, IMPROCEDENTE, etc.  |
| `año_res`                | Año de resolución                                |
| `RES_MATCH_SOURCE`       | Cómo se encontró la resolución: `nro` u `origen` |

---

### 2. Filtrado de filas inválidas

Se descartan filas donde `NRO_EXPEDIENTE` esté vacío. Sin ese campo no hay forma de identificar el expediente y la fila no sirve para ningún análisis ni para alimentar un modelo.

---

### 3. Normalización de fechas

Los campos `FECHA_PRESENTACION_pres` y `FECHA_RESOLUCION` vienen como strings y pueden tener formatos variables dependiendo del Excel de origen (`YYYY-MM-DD`, `DD/MM/YYYY`, `YYYY/MM/DD`). El script los intenta parsear en ese orden y los devuelve siempre en formato estándar `YYYY-MM-DD`.

Si una fecha no puede parsearse con ninguno de los formatos conocidos, se deja vacía y se contabiliza en el reporte final como fecha inválida.

---

### 4. Limpieza de `FORMA_CONCLUSION`

Este campo venía con dos problemas del proceso de carga desde Excel:

- **Prefijo `>>`**: todos los valores estaban precedidos por `>>` (ej. `>>CONFIRMA`, `>>REVOCA`).
- **Valores concatenados**: algunos expedientes tenían más de una forma de conclusión unidas por saltos de línea (ej. `>>CONFIRMA\n>>CONSENTIDO`).

El script elimina el prefijo `>>` y, cuando hay múltiples valores, se queda solo con el primero, que representa la decisión principal de la sala. Resultado: `CONFIRMA`, `REVOCA`, `IMPROCEDENTE`, etc.

---

### 5. Extracción del número de documento

El campo `DOC_DENUNCIADO` venía con el tipo de documento prefijado: `"RUC : 20133840533"` o `"DNI : 12345678"`. El script extrae solo el número (la parte después del `:`), descartando el prefijo textual. Esto lo hace utilizable directamente como identificador numérico.

---

### 6. Normalización de texto libre

Los campos `TIPO_EXPEDIENTE_pres`, `EXPEDIENTE_ORIGEN_pres` y `MATERIA_pres` pueden tener espacios dobles o espacios al inicio/final heredados de los Excel. El script colapsa múltiples espacios a uno solo y aplica trim.

---

### 5. Unificación de variantes ortográficas en `DENUNCIADOS_pres`

Este campo tiene tres problemas de inconsistencia de origen, todos producidos por diferencias entre los Excel de distintos años. El script los resuelve en la función `normalizeDenunciado()`.

**Problema 1 — Tildes:** los Excel de 2017 tenían tildes correctas, los de 2018–2021 no.

**Problema 2 — Puntos en forma legal:** la razón social aparece con y sin puntos indistintamente.

**Problema 3 — Ordinal N°:** el número de comisión aparece con múltiples formatos.

| Original | Normalizado |
|---|---|
| `BANCO DE CRÉDITO DEL PERÚ S.A.` | `BANCO DE CREDITO DEL PERU SA` |
| `BANCO DE CREDITO DEL PERU S.A.` | `BANCO DE CREDITO DEL PERU SA` |
| `SCOTIABANK PERÚ S.A.A.` | `SCOTIABANK PERU SAA` |
| `SCOTIABANK PERU SAA` | `SCOTIABANK PERU SAA` |
| `COMISIÓN DE PROTECCIÓN AL CONSUMIDOR N° 1` | `COMISION DE PROTECCION AL CONSUMIDOR N1` |
| `COMISION DE PROTECCION AL CONSUMIDOR N°1` | `COMISION DE PROTECCION AL CONSUMIDOR N1` |
| `COMISION DE PROTECCIÓN AL CONSUMIDOR N°  1` | `COMISION DE PROTECCION AL CONSUMIDOR N1` |

En conjunto los tres problemas afectan **3,646 filas (~26% del campo)** y generaban **136 grupos de duplicados**. El campo `MATERIA_pres` no tiene ninguno de estos problemas.

---

### 6. Limpieza de los campos de año

Pandas serializa los años como float cuando hay nulos en la columna (ej. `2017.0` en lugar de `2017`). El script elimina el `.0` y valida que el año esté en el rango 2010–2030. Si no cumple, se deja vacío.

---

## Cómo ejecutarlo

```bash
# Asegúrate de tener el CSV en el mismo directorio
go run clean_expedientes.go
```

El output se genera como `expedientes_clean.csv` en el mismo directorio. Al finalizar imprime un reporte:

```
=== Limpieza completada ===
Filas leídas:              14231
Filas sin NRO_EXPEDIENTE:  0 (descartadas)
Filas en output:           14231
Sin resolución (none):     688 (4.8%)
Fechas inválidas:          0
Output guardado en:        expedientes_clean.csv
```

---

## Lo que NO hace este script

- No imputa nulos. Los campos vacíos (materia, denunciado, resolución) se dejan vacíos. El 4.8% de expedientes sin resolución encontrada es una limitación del merge original, no un error de limpieza.
- No desduplicar. Si hay expedientes con el mismo `NRO_EXPEDIENTE`, se conservan tal cual. La lógica de deduplicación corresponde al merge previo en Python.
- No normaliza más allá de tildes. Las variantes de razón social abreviada (`BANCO BBVA PERU` vs `BBVA BANCO CONTINENTAL`, nombre cambiado en 2019) o denunciados múltiples concatenados con `/` no se tocan, ya que requieren lógica de negocio fuera del alcance de esta limpieza.

