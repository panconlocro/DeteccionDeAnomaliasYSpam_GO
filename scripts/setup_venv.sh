#!/usr/bin/env bash
set -euo pipefail

VENV_DIR="${1:-.venv}"
REQUIREMENTS="${2:-requirements.txt}"
KERNEL_NAME="${KERNEL_NAME:-detecciondeanomalias}"
KERNEL_DISPLAY_NAME="${KERNEL_DISPLAY_NAME:-DeteccionDeAnomalias (venv)}"

echo "[setup] Creating venv at '$VENV_DIR'..."
python3 -m venv "$VENV_DIR"

PYTHON_EXE="$VENV_DIR/bin/python"
if [[ ! -x "$PYTHON_EXE" ]]; then
  echo "Python executable not found at $PYTHON_EXE. Is the venv created correctly?" >&2
  exit 1
fi

echo "[setup] Upgrading pip..."
"$PYTHON_EXE" -m pip install --upgrade pip

if [[ -f "$REQUIREMENTS" ]]; then
  echo "[setup] Installing dependencies from '$REQUIREMENTS'..."
  "$PYTHON_EXE" -m pip install -r "$REQUIREMENTS"
else
  echo "[setup] '$REQUIREMENTS' not found; skipping dependency install."
fi

echo "[setup] Registering Jupyter kernel '$KERNEL_NAME'..."
"$PYTHON_EXE" -m ipykernel install --user --name "$KERNEL_NAME" --display-name "$KERNEL_DISPLAY_NAME"

echo "[setup] Done. Activate with: source $VENV_DIR/bin/activate"
echo "[setup] Launch Jupyter with: jupyter lab"
