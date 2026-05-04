# Generacion de data sintetica (Go)

Este documento describe el enfoque usado para crear un dataset sintetico >= 1M filas para el proyecto de deteccion concurrente de spam y anomalias en el libro de reclamaciones.

## Objetivo

- Escalar a >= 1,000,000 registros sin depender de modelos LLM.
- Mantener patrones que permitan detectar spam y anomalias basadas en repeticion de texto y tiempos de envio.
- Ejecutar en una PC estandar (RAM 16 GB / GPU 8 GB) en minutos u horas razonables.

## Fuente base

Se toma como base:

- `data/staging/expedientes_merged.csv`

Este archivo contiene registros reales o preprocesados que sirven como estructura base. El generador replica filas con reemplazo y modifica campos clave para simular nuevos registros.

## Variables sinteticas generadas

El generador crea o sobrescribe las siguientes columnas:

- `DETALLE_QUEJA`: texto sintetico basado en plantillas.
- `HORA_PRESENTACION`: hora sintetica con distribucion realista.
- `ES_SPAM`: bandera binaria (0/1).
- `TIPO_SPAM`: etiqueta del patron de spam o `normal`.

## Logica de generacion

### 1) Muestreo con reemplazo

Para llegar a N filas (por defecto 1,000,000), se seleccionan filas aleatorias de la base y se escriben a salida. Esto permite escalar sin cargar todo en memoria.

### 2) Texto normal por plantillas

Se usa un conjunto de plantillas fijas con componentes aleatorios:

- verbo de accion ("Presento", "Formulo", ...)
- tipo de expediente (de la fila base)
- denunciado (de la fila base o fallback)
- materia (de la fila base o fallback)
- frase de solicitud ("Solicito investigacion...", ...)
- frase extra opcional

Esto produce variedad suficiente para el detector, sin costo de LLM. (se intentó pero por los recursos de la computadora era practicamente imposible son que demore varios dias)

### 3) Hora normal

Se genera `HORA_PRESENTACION` con una distribucion simple:

- 85% entre 08:00 y 17:59
- 15% entre 18:00 y 21:59

## Inyeccion de spam (4 patrones)

Se inyectan patrones que el detector debe identificar:

1) `bombardeo_tiempo` (8%)
   - Se generan grupos de 10 registros con la misma hora y minuto.
   - Textos cortos repetidos.

2) `queja_duplicada` (7%)
   - Se generan grupos de 8 registros con el mismo texto.

3) `rafaga_nocturna` (4%)
   - Horas entre 00:00 y 04:59.

4) `denunciado_fantasma` (3%)
   - Se asigna un denunciado falso por grupos de 5 registros.
   - Hora en rango normal.

Los porcentajes son configurables en el codigo si se requiere ajustar el balance.

## Implementacion

El generador esta en:

- `code/syntheticData.go`

Este script:

- Lee el CSV base.
- Construye el header de salida y asegura columnas requeridas.
- Genera N filas de forma secuencial y las escribe en streaming.

## Uso

```powershell
go run ./code/syntheticData.go -in data/staging/expedientes_merged.csv -out data/synthetic/expedientes_1M.csv -n 1000000
```

Flags disponibles:

- `-in` ruta del CSV base
- `-out` ruta del CSV de salida
- `-n` numero objetivo de filas
- `-seed` semilla RNG

## Validacion rapida

- Verificar que `ES_SPAM` y `TIPO_SPAM` existan y que las proporciones se acerquen a las definidas.
- Revisar que `DETALLE_QUEJA` tenga variedad suficiente.
- Confirmar que los patrones de hora se vean en la distribucion (rafaga nocturna, bombardeo por tiempo).

## Razon del enfoque

Este enfoque cumple con el objetivo del curso:

- Permite generar un dataset grande de forma reproducible y rapida.
- Crea patrones claros de repeticion y tiempos anomales.
- Es compatible con un detector concurrente en Go usando goroutines y mutex para contadores globales.
