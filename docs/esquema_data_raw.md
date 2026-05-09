# Esquema de Datos Raw — Expedientes INDECOPI (2017–2021)

## Descripción General

Los datos crudos provienen de dos fuentes administrativas del INDECOPI (Instituto Nacional de Defensa de la Competencia y de la Protección de la Propiedad Intelectual):

- **Presentados**: expedientes registrados ante la Sala Especializada en Protección al Consumidor (SPC)
- **Resueltos**: expedientes que ya tienen sentencia/resolución

Cada fuente está separada por año (2017–2021) en archivos Excel.

---

## Tabla de Columnas — PRESENTADOS

| Columna | Tipo Dato | Descripción | Ejemplo | Notas |
|---------|-----------|-------------|---------|-------|
| `INGRESO EN SALA` | String | Código o identificador de ingreso a la SPC | `SPC-2017-00001` | Puede variar en formato según año |
| `NRO. EXPEDIENTE ORIGEN` | String | Número del expediente en la comisión regional de origen | `CRC-2016-12345` | Identifica el caso antes de llegar a SPC |
| `TIPO EXPEDIENTE` | String | Categoría del tipo de expediente | `APELACION`, `QUEJA`, `MEDIDA CAUTELAR` | Campo con potencial para standarización |
| `FECHA DE PRESENTACION` | Date/String | Fecha en que ingresó el expediente a la SPC | `2017-03-15` o `15/03/2017` | Formatos inconsistentes según Excel de origen |
| `DENUNCIADO(S)` | String | Nombre de la entidad o persona denunciada | `BANCO DE CRÉDITO DEL PERÚ S.A.` | Puede contener múltiples denunciados; inconsistencias de tildes y puntuación |
| `TIPO Y NUMERO DE DOCUMENTO DENUNCIADO(S)` | String | Identificador único (RUC para empresas, DNI para personas) | `RUC : 20133840533` o `DNI : 12345678` | Formato: `TIPO : NUMERO` con espacios inconsistentes |
| `MATERIA` | String | Asunto o tema del expediente | `TARJETA DE CREDITO`, `SERVICIO FINANCIERO` | Vocabulario controlado; pocas variantes |

---

## Tabla de Columnas — RESUELTOS

| Columna | Tipo Dato | Descripción | Ejemplo | Notas |
|---------|-----------|-------------|---------|-------|
| `NRO. DE EXPEDIENTE` | String | Número único asignado al expediente en SPC | `SPC-2017-00001` | Identificador canónico en la Sala |
| `EXPEDIENTE DE ORIGEN` | String | Número del expediente en la comisión regional de origen | `CRC-2016-12345` | Vincula con el trámite administrativo inicial |
| `TIPO DE EXPEDIENTE` | String | Categoría del tipo de expediente | `APELACION`, `QUEJA`, `MEDIDA CAUTELAR` | Igual rango de valores que en "Presentados" |
| `FECHA DE PRESENTACIÓN` | Date/String | Fecha en que ingresó a la SPC | `2017-03-15` | Formatos inconsistentes (tienes encoding issues en algunos Excel) |
| `DENUNCIADOS` | String | Nombre de la entidad o persona denunciada | `BANCO DE CRÉDITO DEL PERÚ S.A.` | Similar al campo "Presentados"; puede tener múltiples valores |
| `MATERIA SPC` | String | Materia o asunto especializado en SPC | `TARJETA DE CREDITO` | Versión estandarizada de materia |
| `F. RESOLUCIÓN` | Date/String | Fecha en que se emitió la resolución final | `2017-12-20` | Formatos inconsistentes; algunos expedientes sin resolución |
| `NRO. DE RESOLUCIÓN` | String | Número único de la resolución emitida por la Sala | `RES-2017-001234-SPC` | Identificador oficial del documento de sentencia |
| `FORMA DE CONCLUSIÓN` | String | Resultado o decisión del expediente | `>>CONFIRMA`, `>>REVOCA`, `>>IMPROCEDENTE` | Viene con prefijo `>>` y a veces múltiples valores concatenados |

---

## Problemas Identificados en Data Raw

### Presentados
- ❌ Fechas en múltiples formatos (`YYYY-MM-DD`, `DD/MM/YYYY`)
- ❌ Tildes inconsistentes en nombres de denunciados (2017 sí; 2018–2021 no)
- ❌ Puntuación variable en razones sociales (`S.A.`, `S.A`, `SA`)
- ❌ Espacios múltiples y espacios al inicio/final en campos de texto

### Resueltos
- ❌ Fechas en múltiples formatos
- ❌ Prefijo `>>` en `FORMA_CONCLUSION` que no pertenece al dato
- ❌ Múltiples valores concatenados en `FORMA_CONCLUSION` (ej: `>>CONFIRMA\n>>CONSENTIDO`)
- ❌ Tipo de documento con formato inconsistente (`RUC :` vs `RUC:`, espacios variables)
- ❌ Algunos expedientes sin resolución encontrada

---

## Estrategia de Unificación (Merge)

1. **Match por `NRO_EXPEDIENTE`** (Resueltos) ↔ `INGRESO EN SALA` (Presentados)
   - Busca coincidencia directa

2. **Match por `EXPEDIENTE_ORIGEN`**
   - Si no hay match directo, intenta vincular por el número de origen

3. **Resultado**
   - Dataset combinado con sufijos `_pres` y `_res` para campos duplicados
   - Aproximadamente ~14k registros (2017–2021)
   - ~4.8% sin resolución encontrada

---

## Limpieza Posterior (clean_expedientes.go)

Ver [limpieza.md](limpieza.md) para detalles de:
- Selección y eliminación de columnas redundantes
- Normalización de fechas
- Limpieza de `FORMA_CONCLUSION`
- Extracción de documentos
- Normalización de tildes y puntuación en denunciados
- Validación y limpieza de años

