Param(
    [string]$VenvDir = ".venv",
    [string]$Requirements = "requirements.txt",
    [string]$KernelName = "detecciondeanomalias",
    [string]$KernelDisplayName = "DeteccionDeAnomalias (venv)"
)

$ErrorActionPreference = "Stop"

Write-Host "[setup] Creating venv at '$VenvDir'..."
python -m venv $VenvDir

$pythonExe = Join-Path $VenvDir "Scripts\python.exe"
if (!(Test-Path $pythonExe)) {
    throw "Python executable not found at $pythonExe. Is the venv created correctly?"
}

Write-Host "[setup] Upgrading pip..."
& $pythonExe -m pip install --upgrade pip

if (Test-Path $Requirements) {
    Write-Host "[setup] Installing dependencies from '$Requirements'..."
    & $pythonExe -m pip install -r $Requirements
} else {
    Write-Host "[setup] '$Requirements' not found; skipping dependency install."
}

Write-Host "[setup] Registering Jupyter kernel '$KernelName'..."
& $pythonExe -m ipykernel install --user --name $KernelName --display-name $KernelDisplayName

Write-Host "[setup] Done. Activate with: .\$VenvDir\Scripts\Activate.ps1"
Write-Host "[setup] Launch Jupyter with: jupyter lab"
