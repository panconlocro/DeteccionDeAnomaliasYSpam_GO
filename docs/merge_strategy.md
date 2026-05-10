# Merge de Datos: Presentados + Resueltos

## Resumen Ejecutivo

El proceso de merge combina dos fuentes de datos administrativos del INDECOPI (2017–2021):
- **Presentados**: 14,231 expedientes registrados ante la SPC
- **Resueltos**: expedientes con resolución/sentencia

**Resultado**: `expedientes_merged.csv` con 14,231 filas × 18 columnas, donde:
- **95.2%** (13,543 expedientes) tienen resolución encontrada
- **4.8%** (688 expedientes) no tienen resolución registrada

---

## Flujo del Merge

```
┌─────────────────────────────┬──────────────────────────────┐
│   PRESENTADOS (raw/*.xlsx)  │   RESUELTOS (raw/*.xlsx)     │
│   - 7 columnas              │   - 9 columnas               │
│   - 2017-2021               │   - 2017-2021                │
└──────────────┬──────────────┴──────────────┬───────────────┘
               │                             │
               ▼                             ▼
      [Normalizar Nombres]          [Normalizar Nombres]
      [Mapear Columnas]             [Mapear Columnas]
      [Limpiar Espacios]            [Limpiar Espacios]
               │                             │
               ▼                             ▼
      ┌────────────────────────────────────────────┐
      │     MERGE 1: Por NRO_EXPEDIENTE (LEFT)     │
      │  Presentados ◄─ Resueltos (match directo)  │
      │  Resultado: 13,543 matches + 688 sin match │
      └────────────────────┬───────────────────────┘
                           │
                ┌──────────┴──────────┐
                │ ¿Hay expedientes   │
                │ sin resolución?    │
                └──────────┬──────────┘
                           ▼
      ┌────────────────────────────────────────────┐
      │   MERGE 2: Por EXPEDIENTE_ORIGEN (LEFT)    │
      │  Presentados ◄─ Resueltos (match secundario)│
      │  Busca resolver los 688 faltantes          │
      │  Resultado: ~0 matches (expedientes origen  │
      │  no sirve para recuperar resoluciones)      │
      └────────────────────┬───────────────────────┘
                           │
                           ▼
      ┌────────────────────────────────────────────┐
      │  Marca Fuente de Match (RES_MATCH_SOURCE)  │
      │  - "nro": resuelto encontrado por número   │
      │  - "none": sin resolución encontrada       │
      │  Limpia columnas auxiliares                │
      └────────────────────┬───────────────────────┘
                           │
                           ▼
             expedientes_merged.csv
             (14,231 × 18 columnas)
```

---

## Estrategia de Merge Detallada

### Paso 1: Carga y Normalización
- Carga todos los Excel de `data/raw/presentados/` y `data/raw/resueltos/` (2017–2021)
- Normaliza nombres de columnas (espacios → `_`, mayúsculas, acentos removidos)
- Mapea nombres a estándares internos

**Mapeos de Presentados:**
| Columna Original | Columna Estándar |
|---|---|
| `INGRESO EN SALA` | `NRO_EXPEDIENTE` |
| `NRO. EXPEDIENTE ORIGEN` | `EXPEDIENTE_ORIGEN` |
| `TIPO EXPEDIENTE` | `TIPO_EXPEDIENTE` |
| `FECHA DE PRESENTACION` | `FECHA_PRESENTACION` |
| `DENUNCIADO(S)` | `DENUNCIADOS` |
| `TIPO Y NUMERO DE DOCUMENTO DENUNCIADO(S)` | `DOC_DENUNCIADO` |
| `MATERIA` | `MATERIA` |

**Mapeos de Resueltos:**
| Columna Original | Columna Estándar |
|---|---|
| `NRO. DE EXPEDIENTE` | `NRO_EXPEDIENTE` |
| `EXPEDIENTE DE ORIGEN` | `EXPEDIENTE_ORIGEN` |
| `TIPO DE EXPEDIENTE` | `TIPO_EXPEDIENTE` |
| `FECHA DE PRESENTACIÓN` | `FECHA_PRESENTACION` |
| `DENUNCIADOS` | `DENUNCIADOS` |
| `MATERIA SPC` | `MATERIA` |
| `F. RESOLUCIÓN` | `FECHA_RESOLUCION` |
| `NRO. DE RESOLUCION` | `NRO_RESOLUCION` |
| `FORMA DE CONCLUSIÓN` | `FORMA_CONCLUSION` |

### Paso 2: Limpieza de Clave de Merge
```python
# Se aplica .strip() para eliminar espacios al inicio/final
df["NRO_EXPEDIENTE"] = df["NRO_EXPEDIENTE"].astype(str).str.strip()
df["EXPEDIENTE_ORIGEN"] = df["EXPEDIENTE_ORIGEN"].astype(str).str.strip()
```

### Paso 3: Merge Primario (por NRO_EXPEDIENTE)
```python
df_merged = df_presentados.merge(
    df_resueltos,
    on="NRO_EXPEDIENTE",
    how="left",          # Conserva todos los presentados
    suffixes=("_pres", "_res")
)
```

- **Tipo**: LEFT JOIN
- **Clave**: `NRO_EXPEDIENTE` 
- **Sufijos**: `_pres` (presentados) y `_res` (resueltos)
- **Resultado**: 13,543 expedientes con resolución, 688 sin match

### Paso 4: Merge Secundario (por EXPEDIENTE_ORIGEN)
Para intentar resolver los 688 expedientes sin match:
```python
# Prepara tabla de resueltos deduplicada por EXPEDIENTE_ORIGEN
res_by_origen = df_resueltos.drop_duplicates("EXPEDIENTE_ORIGEN", keep="last")
# Renombra columnas con sufijo _res2
res_by_origen = res_by_origen.rename(columns={c: f"{c}_res2" for c in columns})

# Segundo merge
df_merged = df_merged.merge(
    res_by_origen,
    left_on="EXPEDIENTE_ORIGEN_pres",
    right_on="EXPEDIENTE_ORIGEN",
    how="left"
)
```

- **Tipo**: LEFT JOIN
- **Clave**: `EXPEDIENTE_ORIGEN_pres` (presentados) ↔ `EXPEDIENTE_ORIGEN` (resueltos)
- **Sufijo auxiliar**: `_res2` (para diferenciar del primer merge)
- **Resultado**: Cero matches adicionales (los 688 no tienen origen registrado en resueltos)

### Paso 5: Relleno de Nulos (Fill Strategy)
Si el primer merge no encontró resolución pero el segundo sí, rellena:
```python
fill_map = {
    "TIPO_EXPEDIENTE_res": "TIPO_EXPEDIENTE_res2",
    "FECHA_PRESENTACION_res": "FECHA_PRESENTACION_res2",
    # ... (9 campos similares)
}
for target, source in fill_map.items():
    df_merged[target] = df_merged[target].fillna(df_merged[source])
```

### Paso 6: Marca de Fuente (RES_MATCH_SOURCE)
Agrega columna que indica cómo se encontró la resolución:
```python
df_merged["RES_MATCH_SOURCE"] = np.where(
    df_merged["TIPO_EXPEDIENTE_res"].notna(), "nro", "none"
)
# Si no hay match por nro pero sí por origen:
has_res2 = df_merged["TIPO_EXPEDIENTE_res2"].notna()
df_merged.loc[
    (df_merged["RES_MATCH_SOURCE"] == "none") & has_res2,
    "RES_MATCH_SOURCE",
] = "origen"
```

**Posibles valores:**
- `"nro"`: Resolución encontrada match directo por `NRO_EXPEDIENTE` (95.2%)
- `"origen"`: Resolución encontrada por `EXPEDIENTE_ORIGEN` (0% en este dataset)
- `"none"`: No se encontró resolución (4.8%)

### Paso 7: Limpieza Final
- Elimina columnas auxiliares (`EXPEDIENTE_ORIGEN`, columnas `_res2`)
- Mantiene solo las 18 columnas finales

---

## Tabla de Columnas Resultantes

| # | Columna | Fuente | Tipo | Descripción | Valores Nulos |
|---|---------|--------|------|-------------|---|
| 1 | `NRO_EXPEDIENTE` | Presentados | String | Número único del expediente en SPC | 0 (0%) |
| 2 | `EXPEDIENTE_ORIGEN_pres` | Presentados | String | Número del expediente en comisión regional | 25 (0.2%) |
| 3 | `TIPO_EXPEDIENTE_pres` | Presentados | String | Tipo: APELACION, QUEJA, MEDIDA CAUTELAR, etc. | 0 (0%) |
| 4 | `FECHA_PRESENTACION_pres` | Presentados | String | Fecha de ingreso a SPC (formato variable) | 0 (0%) |
| 5 | `DENUNCIADOS_pres` | Presentados | String | Nombre del denunciado (sin normalizar) | 469 (3.3%) |
| 6 | `DOC_DENUNCIADO` | Presentados | String | RUC/DNI del denunciado (formato: TIPO : NUMERO) | 473 (3.3%) |
| 7 | `MATERIA_pres` | Presentados | String | Asunto del caso (ej. TARJETA DE CREDITO) | 2,457 (17.3%) |
| 8 | `año_pres` | Presentados | Integer | Año de presentación | 0 (0%) |
| 9 | `EXPEDIENTE_ORIGEN_res` | Resueltos | String | Número del expediente origen (duplicados) | 829 (5.8%) |
| 10 | `TIPO_EXPEDIENTE_res` | Resueltos | String | Tipo de expediente (duplicado) | 688 (4.8%) |
| 11 | `FECHA_PRESENTACION_res` | Resueltos | String | Fecha de presentación (duplicado) | 688 (4.8%) |
| 12 | `DENUNCIADOS_res` | Resueltos | String | Nombre del denunciado (duplicado) | 1,122 (7.9%) |
| 13 | `MATERIA_res` | Resueltos | String | Materia (duplicado) | 2,863 (20.1%) |
| 14 | `FECHA_RESOLUCION` | Resueltos | String | Fecha de resolución (formato variable) | 688 (4.8%) |
| 15 | `NRO_RESOLUCION` | Resueltos | String | Número oficial de la resolución | 688 (4.8%) |
| 16 | `FORMA_CONCLUSION` | Resueltos | String | Resultado: >>CONFIRMA, >>REVOCA, etc. (sin limpiar) | 688 (4.8%) |
| 17 | `año_res` | Resueltos | Float | Año de resolución | 688 (4.8%) |
| 18 | `RES_MATCH_SOURCE` | Merge | String | Fuente del match: "nro" o "none" | 0 (0%) |

---

## Estadísticas del Merge

### Volumen
```
Total de expedientes presentados:    14,231
  ├─ Con resolución encontrada:     13,543 (95.2%)
  │   └─ Match por NRO_EXPEDIENTE:  13,543 (100%)
  │   └─ Match por EXPEDIENTE_ORIGEN:     0 (0%)
  └─ Sin resolución:                   688 (4.8%)
```

### Cobertura por Año (Resueltos Encontrados)
| Año | Presentados | Con Resueltos | Sin Resueltos | Tasa Cobertura |
|-----|---|---|---|---|
| 2017 | 2,824 | 2,709 | 115 | 95.9% |
| 2018 | 3,018 | 2,869 | 149 | 95.1% |
| 2019 | 3,151 | 3,012 | 139 | 95.6% |
| 2020 | 2,670 | 2,548 | 122 | 95.4% |
| 2021 | 2,568 | 2,405 | 163 | 93.7% |

### Calidad de Datos
| Indicador | Valor | Interpretación |
|---|---|---|
| Filas perdidas en merge | 0 | Se conservan todos los presentados |
| Duplicados por NRO_EXPEDIENTE | 0 | Cada expediente es único |
| Resoluciones duplicadas por origen | 0 | No hay atajos para recuperar datos |
| Tasa de cobertura total | 95.2% | Alta cobertura de resoluciones |

---

## Cómo Usar los Datos Mergeados

1. **Para análisis de presentados puros** → usar columnas `*_pres`
2. **Para análisis de resoluciones** → usar columnas `*_res` (excluir 688 con `RES_MATCH_SOURCE == "none"`)
3. **Para análisis de decisiones judiciales** → usar `FORMA_CONCLUSION` (requiere limpieza previa)
4. **Para serie temporal** → usar `año_pres` y `FECHA_PRESENTACION_pres`
5. **Identificación de entidades** → usar `DOC_DENUNCIADO` (requiere extracción y normalización)

---

## Próximos Pasos

Estos datos mergeados se procesan luego mediante [clean_expedientes.go](limpieza.md) para:
- Eliminar duplicados innecesarios
- Normalizar fechas a formato estándar
- Limpiar `FORMA_CONCLUSION` (remover prefijo `>>`, fusionar múltiples valores)
- Extraer y validar números de documento
- Normalizar nombres de denunciados (tildes, puntuación)

**Salida**: `data/clean/expedientes_clean.csv` (14,231 × 13 columnas, optimizado para análisis)
