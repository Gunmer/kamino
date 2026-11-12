# Fase 4 — Release, rendimiento, documentación y corte

## Contrato

- Asset `kamino-arm64`, `CGO_ENABLED=0`, `-trimpath`, `-s -w`, versión y
  `SelfContained` por `-ldflags`.
- `--version` y `--help` sin cargar catálogo ni configuración de TUI.
- Instalador y actualizador conservan el nombre del asset.
- Python se mueve a `legacy/python/` con `git mv`; no se borra.
- No se publica, no se hace push y no se fusiona a `main`.

## Entregado

- `.goreleaser.yaml`, `scripts/build_release.py` y `make release-snapshot`.
- Workflow de release con Go y el mismo contrato de asset.
- `MIGRATION/benchmark-after.md` con CLI < 30 ms y TUI < 150 ms.
- README, diseño técnico, diseño funcional, AGENTS.md y specs sincronizados.
- Corte a `legacy/python/` y verificación de `legacy/python/src/kamino`.
