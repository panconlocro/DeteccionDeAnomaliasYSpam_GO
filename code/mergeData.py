import pandas as pd
import os

# 1. Cargar y concatenar todos los años de cada tipo
def cargar_todos(carpeta):
    dfs = []
    for archivo in os.listdir(carpeta):
        if archivo.endswith(".xlsx"):
            df = pd.read_excel(f"{carpeta}/{archivo}")
            año = archivo.split(" ")[-1].replace(".xlsx", "")
            df["año"] = año  # columna extra para saber de qué año viene
            dfs.append(df)
    return pd.concat(dfs, ignore_index=True)

df_resueltos = cargar_todos("data/raw/resueltos")
df_presentados = cargar_todos("data/raw/presentados")

# 2. Renombrar columnas para que coincidan antes del join
df_resueltos = df_resueltos.rename(columns={
    "NRO. DE EXPEDIENTE": "NRO_EXPEDIENTE",
    "EXPEDIENTE DE ORIGEN": "EXPEDIENTE_ORIGEN",
    "TIPO DE EXPEDIENTE": "TIPO_EXPEDIENTE",
    "FECHA DE PRESENTACIÓN": "FECHA_PRESENTACION",
    "DENUNCIADOS": "DENUNCIADOS",
    "MATERIA SPC": "MATERIA"
})

df_presentados = df_presentados.rename(columns={
    "INGRESO EN SALA": "NRO_EXPEDIENTE",
    "NRO. EXPEDIENTE ORIGEN": "EXPEDIENTE_ORIGEN",
    "TIPO EXPEDIENTE": "TIPO_EXPEDIENTE",
    "FECHA DE PRESENTACION": "FECHA_PRESENTACION",
    "DENUNCIADO(S)": "DENUNCIADOS",
    "MATERIA": "MATERIA"
})

# 3. Left join por número de expediente
df_merged = df_presentados.merge(df_resueltos, on="NRO_EXPEDIENTE", how="left", suffixes=("_pres", "_res"))

# 4. Guardar
os.makedirs("data/staging", exist_ok=True)
df_merged.to_csv("data/staging/expedientes_merged.csv", index=False)

print(f"Merged: {df_merged.shape[0]} filas, {df_merged.shape[1]} columnas")