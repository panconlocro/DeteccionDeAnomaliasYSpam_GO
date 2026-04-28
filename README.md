## Setup / Data

## Quickstart (Go + Notebooks)

Prerequisites:
- Go 1.21+
- Python 3.10+ (recommended)

### 1) Go (cleaning pipeline)

You need a CSV under `data/` (this folder is gitignored).

Fetch deps:

```powershell
go mod tidy
```

Run the cleaner:

```powershell
go run ./code/clean_csv -in data/complaints-2026-04-14_21_03.enriched.csv -out data/complaints-2026-04-14_21_03.cleaned.csv -qc data/complaints-2026-04-14_21_03.qc.json -dedup=true -workers=8
```

### 2) Python (venv for notebooks)

Windows (PowerShell):

```powershell
./scripts/setup_venv.ps1
./.venv/Scripts/Activate.ps1
jupyter lab
```

macOS/Linux:

```bash
bash scripts/setup_venv.sh
source .venv/bin/activate
jupyter lab
```

If VS Code asks for a kernel, pick: `DeteccionDeAnomalias (venv)`.

The `data/` directory contains a large CSV file and is not included in the repository.

To use this project:
1. Create a `data/` directory in the project root
2. Add your CSV file(s) to the `data/` directory

## Cleaning pipeline (Go)

The cleaner reads a CSV in streaming mode, normalizes values, and outputs a cleaned CSV plus a QC report.
It also removes unused columns created during synthetic data generation.

Detailed documentation: see [docs/cleaning_procedure.md](docs/cleaning_procedure.md).

Example:

```
go run ./code/clean_csv -in data/complaints-2026-04-14_21_03.enriched.csv -out data/complaints-2026-04-14_21_03.cleaned.csv -qc data/complaints-2026-04-14_21_03.qc.json -dedup=true -workers=8
```

Flags:
- `-in` input CSV path
- `-out` output CSV path
- `-qc` QC report JSON path
- `-dedup` enable/disable deduplication
- `-limit` process only the first N rows (0 = all)
- `-workers` number of worker goroutines