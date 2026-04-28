import gdown
import os
import shutil

FOLDER_ID = "1SkL_xy3OBl_sub3-vW5lVFdl98m5-LxG"  

# 1. Descargar todo a una carpeta temporal
os.makedirs("data/raw/temp", exist_ok=True)
os.makedirs("data/raw/resueltos", exist_ok=True)
os.makedirs("data/raw/presentados", exist_ok=True)

gdown.download_folder(FOLDER_ID, output="data/raw/temp/")

# 2. Separar por nombre
for archivo in os.listdir("data/raw/temp"):
    origen = f"data/raw/temp/{archivo}"
    
    if "Resueltos" in archivo:
        shutil.move(origen, f"data/raw/resueltos/{archivo}")
    elif "Presentados" in archivo:
        shutil.move(origen, f"data/raw/presentados/{archivo}")

# 3. Eliminar carpeta temporal
shutil.rmtree("data/raw/temp")

print("Listo!")