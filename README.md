# kamino

CLI para macOS que respalda las configuraciones de las aplicaciones elegidas en **un único
archivo portable** (`tar.gz` con `manifest.json`) y las restaura después, aplicando solo lo que
corresponda a las apps que estén instaladas.

## Estado

MVP v1 implementado: `backup` y `restore` funcionales con catálogo declarativo, detección de
apps, archivado `tar.gz` con manifiesto y checksums, y CLI interactiva.

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

## Distribución

Instalación como herramienta con `uv` (diseño técnico §8):

```bash
uv tool install git+https://github.com/Gunmer/kamino
```

El paquete se publica como `kamino-cli` (el nombre `kamino` ya está ocupado en PyPI) y expone el
comando `kamino`.

## Versionado y releases

El versionado sigue [SemVer](https://semver.org/) y se automatiza con
[python-semantic-release](https://python-semantic-release.readthedocs.io/):

1. Los títulos de los PRs deben seguir Conventional Commits (`feat:`, `fix:`, `docs:`, …); el CI lo valida.
2. Al hacer merge a `main`, se calcula el bump (major/minor/patch) desde los commits, se actualiza
   `CHANGELOG.md`, se crea el tag `vX.Y.Z` y se publica una GitHub Release.
3. La release incluye el binario autocontenido (PyInstaller `onefile`), uno por arquitectura:
   `kamino-arm64` (Apple Silicon) y `kamino-x86_64` (Intel).

Los binarios no están firmados, por lo que macOS les aplica cuarentena al descargarlos. Para usarlos:

```bash
xattr -d com.apple.quarantine ./kamino-arm64
```

El Homebrew tap se publicará tras la primera versión etiquetada.

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
