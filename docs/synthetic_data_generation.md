# Generación de Data Sintética para Simulación de Spam y Anomalías

## 1. Objetivo

El objetivo de este módulo es generar un dataset sintético enriquecido a partir de datos reales de expedientes administrativos, preservando la estructura y distribución general del dataset original mientras se introducen patrones artificiales de spam y anomalías.

El dataset resultante está orientado principalmente a:

* entrenamiento de modelos de Machine Learning,
* detección de spam,
* clasificación de anomalías,
* análisis temporal,
* simulación de comportamiento fraudulento,
* benchmarking de algoritmos.

---

# 2. Enfoque General

La generación sintética sigue un enfoque híbrido:

```text id="j1o2f4"
datos reales
+
transformaciones probabilísticas
+
inyección controlada de anomalías
```

En lugar de generar registros completamente artificiales, el sistema reutiliza filas reales como base estadística y posteriormente añade componentes sintéticos.

Esto permite mantener:

* coherencia estructural,
* relaciones estadísticas reales,
* distribución temporal consistente,
* patrones administrativos plausibles.

---

# 3. Arquitectura del Pipeline

El pipeline completo de generación sigue el siguiente flujo:

```text id="n3k9g8"
Dataset original
        ↓
Muestreo aleatorio de filas reales
        ↓
Construcción del objeto Complaint
        ↓
Generación de timestamp sintético
        ↓
Generación de texto sintético
        ↓
Inyección de anomalías/spam
        ↓
Validación
        ↓
Exportación del dataset enriquecido
```

---

# 4. Preservación de Datos Originales

El sistema conserva todas las columnas originales del dataset fuente.

Ejemplo:

```text id="u6l2x0"
NRO_EXPEDIENTE
TIPO_EXPEDIENTE_pres
DENUNCIADOS_pres
MATERIA_pres
FORMA_CONCLUSION
FECHA_PRESENTACION_pres
...
```

La razón de conservar estas columnas es mantener:

* correlaciones reales,
* distribución estadística original,
* consistencia jurídica y administrativa,
* información contextual útil para modelos predictivos.

Esto permite que el dataset sintético siga representando un entorno cercano al comportamiento real.

---

# 5. Muestreo de Registros Base

El sistema utiliza:

```text id="f5r1a9"
random sampling with replacement
```

Es decir:

* cada registro sintético parte de una fila real seleccionada aleatoriamente,
* una misma fila puede reutilizarse múltiples veces,
* la distribución general del dataset se mantiene relativamente estable.

Este enfoque permite escalar el tamaño del dataset sin perder coherencia estadística.

---

# 6. Construcción del Objeto Complaint

Cada fila seleccionada se transforma en un objeto central llamado:

```text id="t8q0c3"
Complaint
```

Este objeto contiene:

## Datos originales

```text id="b4v8z1"
OriginalData
```

## Datos sintéticos derivados

```text id="b1y2n7"
Timestamp
DetalleQueja
EsSpam
SpamTags
SpamScore
```

El uso de un modelo centralizado facilita la extensibilidad y modularidad del sistema.

---

# 7. Generación Temporal Sintética

## 7.1 Base temporal

El timestamp sintético se deriva directamente de:

```text id="z0h3w2"
FECHA_PRESENTACION_pres
```

Es decir, se preserva:

* año,
* mes,
* día.

Luego se genera una hora sintética adicional.

Ejemplo:

```text id="s2j9e5"
2018-05-03
+
14:22:11
=
2018-05-03 14:22:11
```

---

## 7.2 Distribución horaria

La hora no se genera uniformemente.

El sistema favorece horarios laborales reales, por ejemplo:

* 08:00–18:00 con alta probabilidad,
* horarios nocturnos con menor frecuencia.

Esto busca simular comportamiento humano realista.

---

# 8. Generación de Texto Sintético

El sistema genera dinámicamente la columna:

```text id="h7f4k1"
DETALLE_QUEJA
```

utilizando plantillas probabilísticas.

La generación textual combina:

* acciones,
* entidades,
* materias,
* solicitudes,
* variaciones lingüísticas.

Ejemplo:

```text id="c9l7v3"
"Presento una queja contra Empresa X por problemas relacionados con facturación. Solicito una investigación."
```

---

# 9. Variación Lingüística

Para evitar duplicados exactos y patrones triviales, se introducen variaciones sintéticas:

* cambios de mayúsculas,
* eliminación de tildes,
* signos adicionales,
* pequeñas alteraciones léxicas,
* ruido ortográfico ligero.

Ejemplo:

```text id="w5e0t2"
Solicito atención inmediata.
SOLICITO ATENCION INMEDIATA!!
Requiero atención inmediata urgente.
```

Esto mejora el realismo del dataset y dificulta la detección basada únicamente en coincidencias exactas.

---

# 10. Inyección de Anomalías y Spam

El sistema implementa un motor modular de anomalías.

Cada anomalía representa un patrón específico de comportamiento sospechoso o fraudulento.

Las anomalías se aplican probabilísticamente sobre cada registro.

---

# 11. Tipos de Spam Implementados

## 11.1 Temporal Burst Spam

### Objetivo

Simular ráfagas coordinadas de actividad.

### Técnica

Los timestamps de múltiples registros son concentrados en intervalos de tiempo muy cortos.

### Representa

* campañas automatizadas,
* spam masivo,
* actividad coordinada.

---

## 11.2 Duplicate Text Spam

### Objetivo

Simular reclamos repetidos o semiduplicados.

### Técnica

Se reutiliza un texto base aplicando pequeñas modificaciones sintéticas.

### Ejemplo

```text id="l1g6r9"
Solicito solución inmediata.
SOLICITO SOLUCION INMEDIATA!!
Requiero solución inmediata urgente.
```

### Representa

* bots,
* mensajes automatizados,
* campañas repetitivas.

---

## 11.3 Night Activity Spam

### Objetivo

Simular actividad sospechosa en horarios atípicos.

### Técnica

El timestamp es desplazado hacia horarios nocturnos.

### Representa

* automatización,
* tráfico no humano,
* comportamiento anómalo.

---

## 11.4 Fake Entity Spam

### Objetivo

Simular entidades inexistentes o sospechosas.

### Técnica

La columna:

```text id="j3w8f0"
DENUNCIADOS_pres
```

es reemplazada por entidades sintéticas plausibles.

### Ejemplo

```text id="u0p7m6"
Corporación Integral SAC
Grupo Financiero Andino EIRL
Servicios Comerciales SA
```

### Representa

* empresas ficticias,
* entidades fraudulentas,
* actores inexistentes.

---

# 12. Combinación de Anomalías

Las anomalías no son mutuamente excluyentes.

Un mismo registro puede contener múltiples tipos de spam simultáneamente.

Ejemplo:

```text id="d5v2c8"
duplicate_text
+
night_activity
+
fake_entity
```

Esto busca representar escenarios más realistas, ya que el spam real suele presentar múltiples señales simultáneas.

---

# 13. Etiquetado del Dataset

Cada registro contiene:

## ES_SPAM

Indica si el registro contiene anomalías.

```text id="h2y4n1"
0 = normal
1 = spam
```

---

## SPAM_TAGS

Lista de anomalías aplicadas.

Ejemplo:

```text id="u9l1w7"
duplicate_text;night_activity
```

---

## SPAM_SCORE

Puntaje probabilístico sintético de sospecha.

Valores altos representan mayor nivel de anomalía.

---

# 14. Validación

Antes de exportar cada registro, el sistema ejecuta validaciones básicas:

* texto no vacío,
* entidad válida,
* consistencia estructural,
* integridad mínima del registro.

Esto evita generar datos corruptos o inconsistentes.

---

# 15. Exportación Final

El dataset final conserva todas las columnas originales y añade nuevas columnas sintéticas:

```text id="n6c0f5"
TIMESTAMP
DETALLE_QUEJA
ES_SPAM
SPAM_TAGS
SPAM_SCORE
```

El resultado es un dataset híbrido orientado a simulación de eventos y detección de anomalías.

---

# 16. Tipo de Generación Utilizada

El enfoque implementado corresponde principalmente a:

```text id="m8e2r0"
Rule-Based Synthetic Data Generation
```

combinado con:

```text id="p4t7z1"
Probabilistic Behavioral Simulation
```

El sistema no utiliza:

* redes neuronales,
* GANs,
* modelos generativos profundos,
* LLMs,
* embeddings.

En su lugar, utiliza:

* reglas heurísticas,
* generación probabilística,
* simulación temporal,
* transformaciones textuales,
* anomalías modulares.

---

# 17. Ventajas del Enfoque

## Preserva realismo estadístico

Al reutilizar datos reales como base.

---

## Es extensible

Nuevas anomalías pueden añadirse fácilmente.

---

## Permite trazabilidad

Cada anomalía aplicada queda registrada explícitamente.

---

## Facilita experimentación

Permite generar datasets de distintos tamaños y niveles de complejidad.

---

# 18. Limitaciones

## El texto sintético sigue siendo heurístico

No posee comprensión semántica profunda.

---

## Las anomalías son programadas manualmente

No emergen automáticamente desde los datos.

---

## No replica completamente comportamiento humano real

Aunque intenta aproximarlo estadísticamente.

---

# 19. Conclusión

El sistema implementa una arquitectura modular para generación de data sintética orientada a simulación de spam y anomalías sobre datos tabulares reales.

El enfoque combina:

* preservación de estructura real,
* generación probabilística,
* simulación temporal,
* enriquecimiento textual,
* inyección controlada de anomalías.

El resultado es un dataset híbrido adecuado para experimentación en Machine Learning, análisis de comportamiento anómalo y entrenamiento de modelos de detección de spam.
