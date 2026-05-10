# Informe Técnico — Detección de Anomalías y Spam (Go + Promela/SPIN)

**Repositorio:** `panconlocro/DeteccionDeAnomaliasYSpam_GO`  
**Fecha:** 2026-05-10

---

## 1. Calidad de código

### Hallazgos principales

| Área | Observación | Evidencia |
|---|---|---|
| Organización | Estructura modular correcta en `cmd/` + `internal/` | `code/Logistic_Rregression/{cmd,internal}` |
| Legibilidad | En `generateData` hay mucho código legado comentado que reduce claridad | `code/generateData/cmd/main.go`, `code/generateData/internal/anomly/*.go` |
| Nombres | Hay typos en nombres de carpetas/paquetes (`Logistic_Rregression`, `anomly`) | Árbol de directorios |
| Duplicación | Normalización de acentos repetida en varios paquetes | `clean_csv`, `text`, `features/categorical.go` |
| Complejidad | `generateData/cmd/main.go` concentra demasiada orquestación en `main` | `code/generateData/cmd/main.go` |

### Evaluación

- El módulo de clasificación (`Logistic_Rregression`) está bien separado por responsabilidades.
- El módulo de generación sintética funciona, pero su mantenibilidad es menor por deuda técnica visible (comentarios masivos, formateo inconsistente y acoplamiento en `main`).

---

## 2. Concurrencia

### Uso de goroutines y sincronización

| Componente | Patrón | Evaluación |
|---|---|---|
| Preprocesamiento | Worker pool con `jobs chan int` + `WaitGroup` | Correcto |
| Vectorización | Worker pool por índices + `WaitGroup` | Correcto |
| DF / categorías | Mapa local por worker + merge final | Correcto y seguro |
| Entrenamiento | Gradiente local por worker + reduce en hilo principal | Correcto (sin write races en pesos) |
| Evaluación | Métricas locales + merge con `sync.Mutex` | Correcto |

### Riesgos

- **Race conditions:** no se observan races directas en la ruta concurrente principal del clasificador.
- **Deadlocks:** no se observan deadlocks en los patrones de canal usados.
- **Cuellos de botella:** reducción de gradientes y actualización de pesos siguen siendo seriales; normal por diseño.
- **Punto de mejora:** `GenerateContext.mu` existe pero no se usa realmente en `generateData` (inconsistencia de diseño).

---

## 3. Rendimiento

### Diferencias secuencial vs concurrente

- El proyecto implementa comparación formal con `-mode compare` y reportes por etapa.
- Se mide tanto tiempo total como tiempo sin lectura CSV (correcto para aislar speedup de cómputo).

### Hallazgos

- `training` y `vectorization` son etapas dominantes del tiempo total.
- Carga de CSV completa en memoria limita escalabilidad para datasets aún más grandes.
- El uso de vectores dispersos (`SparseVector`) es una buena decisión para memoria/CPU.

### Speedup

- La metodología de `trimmed mean` es apropiada para reducir outliers.
- El enfoque concurrente está bien planteado para escalar por workers, con límites esperables por Amdahl y ancho de banda de memoria.

---

## 4. Seguridad y robustez

### Hallazgos

| Tema | Hallazgo |
|---|---|
| Validación | Buenas validaciones básicas de columnas y labels en loader de clasificación |
| Manejo de recursos | Hay lugares donde `Flush()` no verifica error |
| Robustez I/O | En `writer.NewCSVWriter`, si falla la escritura inicial del header no se cierra el archivo |
| Configuración | En `clean_csv` y `generateData` hay rutas hardcodeadas (menos robusto para ejecución flexible) |

---

## 5. Verificación formal (Promela/SPIN)

Archivo analizado: `code/promela/ModeladoInicial.pml`.

### Evaluación

| Punto | Estado |
|---|---|
| Modelo concurrente básico con workers | Presente |
| Propiedades LTL (`liveness`, `safety`) | Presentes |
| Relación conceptual con Go | Parcialmente alineada (acumulación local + suma global) |
| Cobertura del pipeline real | Limitada (modelo simplificado) |

### Limitaciones del modelo

- El modelo no representa la arquitectura real por etapas (preprocess, vocab, vectorization, training, evaluation).
- Las propiedades LTL actuales son válidas pero poco exigentes para correctitud funcional completa.
- No hay evidencia en repo de corrida SPIN automatizada ni trazas de verificación.

---

## 6. GAPs identificados

### Resumen por dimensión

| Dimensión | GAP principal |
|---|---|
| Calidad | Código legado comentado y nomenclatura inconsistente |
| Arquitectura | Exceso de lógica en `main` de `generateData` |
| Seguridad/robustez | Manejo incompleto de errores de flush/cierre |
| Concurrencia | Buen núcleo concurrente, pero generación sintética no explota concurrencia |
| Rendimiento | Carga completa en memoria; no hay streaming |
| Testing | Cobertura limitada en módulos fuera del clasificador |
| Escalabilidad | Dependencia fuerte de RAM y rutas rígidas |
| Formal | Modelo Promela simplificado frente al sistema real |

---

## 7. Recomendaciones

1. **Eliminar código comentado legado** en `generateData` y `anomly`.
2. **Refactorizar `generateData/cmd/main.go`** en servicios por etapa (orquestador + componentes).
3. **Centralizar normalización de texto/acentos** en utilidad compartida.
4. **Corregir manejo de recursos** (`Flush` con chequeo de error y cierre en errores tempranos).
5. **Agregar flags CLI** para rutas, cantidad de filas y seed en binarios con hardcode actual.
6. **Ampliar tests** en `generateData` y pruebas de concurrencia/reproducibilidad.
7. **Evolucionar modelo Promela** para acercarlo al pipeline real y fortalecer propiedades LTL.

---

## 8. Conclusión

El proyecto tiene una base sólida en concurrencia aplicada al pipeline de clasificación (Go, goroutines, `sync.Mutex`, reducción de gradientes por workers), y una documentación académica útil.  

Sus principales debilidades están en mantenibilidad y robustez operativa del módulo de generación de datos (deuda técnica visible, manejo de errores mejorable y configuración rígida).  

Desde una perspectiva profesional-académica: **muy buen enfoque concurrente en el núcleo del clasificador**, con margen claro de mejora en higiene de código, pruebas integrales y profundidad de verificación formal.

