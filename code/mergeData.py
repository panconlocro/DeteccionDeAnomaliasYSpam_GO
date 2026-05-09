import pandas as pd
import numpy as np
import os
import re
import unicodedata

def normalize_col(name):
    name = str(name)
    name = unicodedata.normalize("NFKD", name)
    name = "".join(c for c in name if not unicodedata.combining(c))
    name = name.upper()
    name = re.sub(r"[^A-Z0-9]+", "_", name).strip("_")
    return name


def rename_by_map(df, mapping):
    normalized = {normalize_col(c): c for c in df.columns}
    rename = {}
    for norm, target in mapping.items():
        if norm in normalized:
            rename[normalized[norm]] = target
    return df.rename(columns=rename)


def parse_year(filename):
    match = re.search(r"(\d{4})", filename)
    return int(match.group(1)) if match else None


def cargar_todos(carpeta):
    dfs = []
    for archivo in os.listdir(carpeta):
        if not archivo.endswith(".xlsx"):
            continue
        df = pd.read_excel(os.path.join(carpeta, archivo))
        df["año"] = parse_year(archivo)
        dfs.append(df)
    return pd.concat(dfs, ignore_index=True)

df_resueltos = cargar_todos("data/raw/resueltos")
df_presentados = cargar_todos("data/raw/presentados")

presentados_map = {
    "INGRESO_EN_SALA": "NRO_EXPEDIENTE",
    "NRO_EXPEDIENTE_ORIGEN": "EXPEDIENTE_ORIGEN",
    "TIPO_EXPEDIENTE": "TIPO_EXPEDIENTE",
    "FECHA_DE_PRESENTACION": "FECHA_PRESENTACION",
    "DENUNCIADO_S": "DENUNCIADOS",
    "DENUNCIADOS": "DENUNCIADOS",
    "TIPO_Y_NUMERO_DE_DOCUMENTO_DENUNCIADO_S": "DOC_DENUNCIADO",
    "MATERIA": "MATERIA",
}

resueltos_map = {
    "NRO_DE_EXPEDIENTE": "NRO_EXPEDIENTE",
    "EXPEDIENTE_DE_ORIGEN": "EXPEDIENTE_ORIGEN",
    "TIPO_DE_EXPEDIENTE": "TIPO_EXPEDIENTE",
    "FECHA_DE_PRESENTACION": "FECHA_PRESENTACION",
    "DENUNCIADOS": "DENUNCIADOS",
    "MATERIA_SPC": "MATERIA",
    "F_RESOLUCION": "FECHA_RESOLUCION",
    "NRO_DE_RESOLUCION": "NRO_RESOLUCION",
    "FORMA_DE_CONCLUSION": "FORMA_CONCLUSION",
}

df_presentados = rename_by_map(df_presentados, presentados_map)
df_resueltos = rename_by_map(df_resueltos, resueltos_map)

for col in ["NRO_EXPEDIENTE", "EXPEDIENTE_ORIGEN"]:
    if col in df_presentados:
        df_presentados[col] = df_presentados[col].astype(str).str.strip()
    if col in df_resueltos:
        df_resueltos[col] = df_resueltos[col].astype(str).str.strip()

df_merged = df_presentados.merge(
    df_resueltos,
    on="NRO_EXPEDIENTE",
    how="left",
    suffixes=("_pres", "_res"),
)

res_by_origen = df_resueltos.copy()
if "año" in res_by_origen.columns:
    res_by_origen["año"] = pd.to_numeric(res_by_origen["año"], errors="coerce")
res_by_origen = (
    res_by_origen.sort_values(["EXPEDIENTE_ORIGEN", "año"])
    .drop_duplicates("EXPEDIENTE_ORIGEN", keep="last")
)
res_by_origen = res_by_origen.rename(
    columns={
        c: f"{c}_res2" for c in res_by_origen.columns if c != "EXPEDIENTE_ORIGEN"
    }
)

df_merged = df_merged.merge(
    res_by_origen,
    left_on="EXPEDIENTE_ORIGEN_pres",
    right_on="EXPEDIENTE_ORIGEN",
    how="left",
)

res_cols = [
    "EXPEDIENTE_ORIGEN_res",
    "TIPO_EXPEDIENTE_res",
    "FECHA_PRESENTACION_res",
    "DENUNCIADOS_res",
    "MATERIA_res",
    "año_res",
    "FECHA_RESOLUCION",
    "NRO_RESOLUCION",
    "FORMA_CONCLUSION",
]

for col in res_cols:
    if col in df_merged:
        df_merged[col] = df_merged[col].replace("", pd.NA)

fill_map = {
    "EXPEDIENTE_ORIGEN_res": "EXPEDIENTE_ORIGEN_res2",
    "TIPO_EXPEDIENTE_res": "TIPO_EXPEDIENTE_res2",
    "FECHA_PRESENTACION_res": "FECHA_PRESENTACION_res2",
    "DENUNCIADOS_res": "DENUNCIADOS_res2",
    "MATERIA_res": "MATERIA_res2",
    "año_res": "año_res2",
    "FECHA_RESOLUCION": "FECHA_RESOLUCION_res2",
    "NRO_RESOLUCION": "NRO_RESOLUCION_res2",
    "FORMA_CONCLUSION": "FORMA_CONCLUSION_res2",
}

for target, source in fill_map.items():
    if target in df_merged and source in df_merged:
        df_merged[target] = df_merged[target].fillna(df_merged[source])

df_merged["RES_MATCH_SOURCE"] = np.where(
    df_merged["TIPO_EXPEDIENTE_res"].notna(), "nro", "none"
)
# Si no hay match por nro pero sí por origen:
has_res2 = df_merged["TIPO_EXPEDIENTE_res2"].notna()
df_merged.loc[
    (df_merged["RES_MATCH_SOURCE"] == "none") & has_res2,
    "RES_MATCH_SOURCE",
] = "origen"

if "EXPEDIENTE_ORIGEN" in df_merged.columns:
    df_merged = df_merged.drop(columns=["EXPEDIENTE_ORIGEN"])
res2_cols = [c for c in df_merged.columns if c.endswith("_res2")]
if res2_cols:
    df_merged = df_merged.drop(columns=res2_cols)

# 4. Guardar
os.makedirs("data/staging", exist_ok=True)
df_merged.to_csv("data/staging/expedientes_merged.csv", index=False)

matched_nro = (df_merged["RES_MATCH_SOURCE"] == "nro").sum()
matched_origen = (df_merged["RES_MATCH_SOURCE"] == "origen").sum()

print(f"Merged: {df_merged.shape[0]} filas, {df_merged.shape[1]} columnas")
print(f"Matches por NRO_EXPEDIENTE: {matched_nro}")
print(f"Matches por EXPEDIENTE_ORIGEN: {matched_origen}")