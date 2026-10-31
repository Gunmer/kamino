# Release automático por semver

**Objetivo:** al mergear a `main`, incrementar la versión siguiendo semver, generar el binario, actualizar el changelog, taggear la versión y crear la GitHub Release.
**Rama:** `feature/mvp` · **Delivery:** `ask-on-risk` · **Test runner:** `uv run pytest`, `uv run ruff check .` (verificación local: `python -m kamino --version`, build PyInstaller, `semantic-release version --noop`).

## Specs

- **S1** — "cuando una PR se mergee en main, se incremente la version siguiendo semver": el bump se calcula automáticamente a partir de los commits convencionales desde el último tag.
- **S2** — "se genere el binario": binario autocontenido (PyInstaller `onefile`), una build por arquitectura (arm64 + x86_64), como prevé `docs/Diseño técnico.md` §8.
- **S3** — "actualice el changelog": `CHANGELOG.md` generado y mantenido automáticamente.
- **S4** — "tagee la version y se cree la release para github": tag `vX.Y.Z` + GitHub Release con los binarios adjuntos.
- **S5** — Decisión "Automático (Conventional Commits)": `python-semantic-release` decide el bump; el repo ya usa commits convencionales.
- **S6** — Decisión "Sin protección (o permito el push)": el workflow pushea el commit de versión y el tag con `GITHUB_TOKEN` (`contents: write`).

Supuestos (no pedidos explícitamente): semver estricto pre-1.0 (`major_on_zero = false`, un `BREAKING CHANGE` en `0.x` sube `minor`); firma/notarización del binario queda fuera de alcance (solo se documenta la cuarentena).

## Tasks

- **T1** — pyproject.toml: config `[tool.semantic_release]` + dep `python-semantic-release` + extra `build` con `pyinstaller`. [S1,S2,S3,S5] · inline · hecha
- **T2** — `src/kamino/__main__.py`: entry para PyInstaller (y `python -m kamino`). [S2] · inline · hecha
- **T3** — `.github/workflows/release.yml`: bump+changelog+tag, build de binarios por arquitectura, GitHub Release. [S1,S2,S3,S4,S6] · inline · hecha
- **T4** — `.github/workflows/ci.yml`: lint de título de PR (Conventional Commits). [S5] · inline · hecha
- **T5** — `CHANGELOG.md` bootstrap + docs (README + diseño técnico §8: flujo y cuarentena). [S3,S4] · inline · hecha

## Log

- **L1** — "quiero que cuando una PR se mergee en main, se incremente la version siguiendo semver, se genere el binario, actualice el changelog y tagee la version y se cree la release para github."
- **L2** (2026-10-31) — El usuario eligió "Automático (Conventional Commits)" como enfoque de versionado.
- **L3** (2026-10-31) — El usuario confirmó que `main` no tiene protección que bloquee el push del workflow.
- **L4** — Espejo Engram pendiente: no hay herramienta `mem_*` inyectada en esta sesión; solo persiste la copia local `odd/tasks/auto-release.md`.
- **L5** — `python-semantic-release` resolvió a v10.7.0 (`>=9.12`). v10 exige tag previo: sin tags calcula "initial release" (no lee `0.1.0` de pyproject). Config: `allow_zero_version = true`, `major_on_zero = false`.
- **L6** — Verificación: `pytest` 191 passed, `ruff check` OK, `python -m kamino --version` → `kamino 0.1.0`, binario PyInstaller compilado y `--version` OK, `semantic-release --noop version` en feature/mvp → `0.1.0`.
- **L7** — PSR v10 `version` también crea la release en el VCS; se añadió `--no-vcs-release` para que `softprops/action-gh-release` la cree con los binarios adjuntos.
