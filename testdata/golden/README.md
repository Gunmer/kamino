# Fixtures doradas

Estos archivos se generan desde la implementación Python con
`scripts/gen_golden.py` (o `make golden`) usando HOMEs sintéticos. No se editan
a mano.

- `clone.tar.gz`: clon canónico con `manifest.json` primero. El generador
  normaliza mtime, uid/gid, nombres de usuario y la cabecera gzip para que dos
  ejecuciones produzcan los mismos bytes.
- `manifest.json`: manifiesto validado por Python, con el orden de campos y los
  `null` explícitos de Pydantic.
- `plan.json`: operaciones, omisiones y avisos del plan.
- `conflicts.json`: hunks, decisiones y composiciones de un conflicto de texto.
- `texts.json`: ayuda, glosario y plantillas visibles.
- `codes.json`: códigos de salida 0, 1, 2 y 3 de la CLI Python.
- `SHA256SUMS`: resumen de integridad de los fixtures.

Go compara el modelo semántico (nombres, tipos, contenidos, enlaces, modos y
categorías), no los bytes del tar.gz de Python. `scripts/guard.sh` impide
cambiar estos archivos sin una decisión registrada en `MIGRATION/DECISIONS.md`.
