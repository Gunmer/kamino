# kamino (implementación Python)

Implementación histórica de referencia de `kamino-cli`, conservada para la
matriz de compatibilidad con la implementación Go. La implementación activa y
la documentación principal están en la raíz del repositorio.

## Desarrollo

Desde este directorio:

```bash
uv sync --all-extras --dev
uv run pytest
```

La CLI se ejecuta con `uv run python -m kamino` o, tras sincronizar, con el
ejecutable `kamino` del entorno virtual.
