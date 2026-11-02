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

1. Los commits integrados en `main` deben seguir Conventional Commits. Con merge tradicional o
   rebase se analizan los commits de la rama; con squash, conserva un mensaje Conventional Commit
   adecuado en el commit resultante. CI no valida los títulos de los PRs.
2. Después de que CI pase en `main`, se calcula la siguiente versión desde el último tag:
   `feat:` aumenta minor, `fix:` aumenta patch y `!` o un footer `BREAKING CHANGE:` identifica
   cambios incompatibles. Si solo hay `docs:`, `ci:`, `chore:`, `perf:` u otros cambios sin ruptura
   de compatibilidad, no se genera versión ni release.
3. Durante las versiones `0.x`, los cambios incompatibles aumentan minor (`major_on_zero = false`).
   Desde `1.0.0`, aumentan major. Se toma el mayor incremento entre los commits pendientes.
4. Se actualizan `pyproject.toml`, `src/kamino/__init__.py`, `uv.lock` y `CHANGELOG.md`. El binario
   se construye y comprueba antes de empujar el commit de release y el tag `vX.Y.Z`.
5. Se crea una GitHub Release con las notas y el binario autocontenido `kamino-arm64`
   (PyInstaller `onefile`, Apple Silicon), incluido el catálogo de aplicaciones. No se distribuye
   actualmente un binario Intel. La release permanece en borrador hasta subir el binario.

Por ejemplo, desde `0.1.0`, `fix: corrige la restauración` produce `0.1.1`, mientras que
`feat: añade una aplicación` o `feat!: cambia el formato` produce `0.2.0`. Un merge que solo
contenga `ci: ajusta el workflow` no publica nada.

Las publicaciones se serializan. Si `main` avanza durante la construcción, se cancela el push
sin forzarlo; la ejecución del nuevo commit validado retomará los cambios pendientes. Los pushes
y tags del bot no disparan otro workflow: todo el proceso ocurre en **Release**.

Si falla la publicación después de empujar el tag, ejecuta manualmente **Release** desde la rama
`main` e indica ese tag en el campo `tag`. Esta recuperación reconstruye y comprueba la misma
versión y completa la release sin incrementar la versión, duplicar el changelog ni reemplazar
assets ya subidos. Si GitHub dejó un asset vacío por una subida fallida, lo elimina antes de
reintentar. No permite publicar tags ajenos a un commit de release integrado en `main`.

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
