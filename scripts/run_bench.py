#!/usr/bin/env python3
import subprocess
import json
import os
import sys
import time
import threading
import psutil
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RESULTS = ROOT / 'data' / 'bench'
RESULTS.mkdir(parents=True, exist_ok=True)

runs    = 20
workers = 4

concurrent_out = RESULTS / 'concurrente.json'
secuencial_out = RESULTS / 'secuencial.json'
summary_out    = RESULTS / 'summary.json'


def monitor_process(proc, interval=0.2):
    """
    Monitorea CPU% y memoria RSS de un proceso mientras corre.
    Devuelve (lista de muestras cpu, lista de muestras mem_mb).
    """
    cpu_samples = []
    mem_samples = []
    try:
        ps = psutil.Process(proc.pid)
        while proc.poll() is None:
            try:
                cpu = ps.cpu_percent(interval=interval)
                mem = ps.memory_info().rss / (1024 * 1024)  # MB
                cpu_samples.append(cpu)
                mem_samples.append(mem)
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                break
    except psutil.NoSuchProcess:
        pass
    return cpu_samples, mem_samples


def run_and_monitor(cmd, label):
    """
    Lanza un comando, lo monitorea en un hilo aparte,
    y devuelve métricas de recursos.
    """
    print(f'\nEjecutando {label}...')
    proc = subprocess.Popen(cmd, stdout=sys.stdout, stderr=sys.stderr)

    cpu_samples, mem_samples = [], []

    monitor_thread = threading.Thread(
        target=lambda: cpu_samples_mem_fill(proc, cpu_samples, mem_samples)
    )
    monitor_thread.start()
    proc.wait()
    monitor_thread.join()

    if proc.returncode != 0:
        print(f'Error ejecutando {label}', file=sys.stderr)
        sys.exit(proc.returncode)

    avg_cpu = sum(cpu_samples) / len(cpu_samples) if cpu_samples else 0
    max_cpu = max(cpu_samples) if cpu_samples else 0
    avg_mem = sum(mem_samples) / len(mem_samples) if mem_samples else 0
    max_mem = max(mem_samples) if mem_samples else 0

    return {
        'avg_cpu_percent': round(avg_cpu, 2),
        'max_cpu_percent': round(max_cpu, 2),
        'avg_mem_mb':      round(avg_mem, 2),
        'max_mem_mb':      round(max_mem, 2),
        'samples':         len(cpu_samples),
    }


def cpu_samples_mem_fill(proc, cpu_samples, mem_samples):
    """Función auxiliar que llena las listas desde el hilo monitor."""
    interval = 0.2
    try:
        ps = psutil.Process(proc.pid)
        while proc.poll() is None:
            try:
                cpu = ps.cpu_percent(interval=interval)
                mem = ps.memory_info().rss / (1024 * 1024)
                cpu_samples.append(cpu)
                mem_samples.append(mem)
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                break
    except psutil.NoSuchProcess:
        pass


# --- Ejecutar concurrente ---
recursos_conc = run_and_monitor(
    ['go', 'run', './code/Deteccion_Concurrente',
     f'-runs={runs}', f'-workers={workers}', f'-out={concurrent_out}'],
    'detector concurrente'
)

# --- Ejecutar secuencial ---
recursos_seq = run_and_monitor(
    ['go', 'run', './code/Deteccion_Secuencial',
     f'-runs={runs}', f'-out={secuencial_out}'],
    'detector secuencial'
)

# --- Leer JSONs de resultados ---
with open(concurrent_out, 'r', encoding='utf-8') as f:
    conc = json.load(f)
with open(secuencial_out, 'r', encoding='utf-8') as f:
    seq = json.load(f)

# --- Calcular speedup ---
conc_time = conc.get('media_recortada_seconds') or conc.get('promedio_seconds')
seq_time  = seq.get('media_recortada_seconds')  or seq.get('promedio_seconds')

speedup = None
if conc_time and seq_time:
    try:
        speedup = round(float(seq_time) / float(conc_time), 4)
    except Exception:
        speedup = None

# --- Calcular eficiencia (speedup / workers) ---
eficiencia = None
if speedup is not None:
    eficiencia = round(speedup / workers, 4)

# --- Armar summary ---
summary = {
    'concurrente': {
        **conc,
        'recursos': recursos_conc,
    },
    'secuencial': {
        **seq,
        'recursos': recursos_seq,
    },
    'speedup_media_recortada': speedup,
    'eficiencia_paralela':     eficiencia,   # speedup / workers, ideal = 1.0
    'workers_used':            workers,
    'runs':                    runs,
}

with open(summary_out, 'w', encoding='utf-8') as f:
    json.dump(summary, f, indent=2, ensure_ascii=False)

print('\n===== RESUMEN =====')
print(f"Speedup (media recortada): {speedup}x")
print(f"Eficiencia paralela:       {eficiencia} (ideal 1.0 con {workers} workers)")
print(f"CPU promedio  — secuencial:  {recursos_seq['avg_cpu_percent']}%  |  concurrente: {recursos_conc['avg_cpu_percent']}%")
print(f"Memoria pico  — secuencial:  {recursos_seq['max_mem_mb']} MB  |  concurrente: {recursos_conc['max_mem_mb']} MB")
print(f'\nResumen escrito en {summary_out}')
print('Hecho.')