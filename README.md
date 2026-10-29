# kamino

CLI para macOS que respalda las configuraciones de las aplicaciones elegidas en **un único
archivo portable** (`tar.gz` con `manifest.json`) y las restaura después, aplicando solo lo que
corresponda a las apps que estén instaladas.

## Estado

Esqueleto del proyecto (paso 1 del plan de implementación del diseño técnico). Los comandos
`backup` y `restore` aún no están implementados.

## Requisitos

- Python 3.11 o superior (mínimo 3.11.4 por `tarfile` con `filter="data"`).
- macOS.

## Instalación (desarrollo)

```bash
uv sync --all-extras
uv run kamino --help
```

Sin `uv`, con `pip`:

```bash
python3.11 -m venv .venv
.venv/bin/pip install -e ".[dev]"
.venv/bin/kamino --help
```

## Uso

```bash
kamino --help
kamino backup --apps vscode,iterm2,git
kamino restore kamino-<equipo>-<fecha>.tar.gz
```

## Desarrollo

```bash
uv run pytest          # pruebas
uv run ruff check .    # lint
uv run ruff format .   # formato
```

## Estructura

```
src/kamino/
├── cli.py            # comandos backup y restore
├── ui.py             # selección interactiva, tablas, progreso
├── backup.py         # caso de uso: plan y ejecución del respaldo
├── restore.py        # caso de uso: plan y ejecución de la restauración
├── plan.py           # Plan y Operation, compartidos
├── catalog/
│   ├── models.py     # AppDefinition (validación)
│   ├── loader.py     # catálogo embebido y del usuario
│   └── apps/*.toml   # una definición por app
├── detect.py         # ¿está instalada? ¿qué versión?
├── archive.py        # tar.gz, manifiesto, checksums
├── paths.py          # normalización relativa al home
├── snapshot.py       # copia previa a restaurar
└── errors.py         # excepciones propias
```

Los diseños de referencia están en `docs/` (`Diseño funcional.md` y `Diseño técnico.md`).
