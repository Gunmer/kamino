"""Configuración de la TUI en `~/.config/kamino/config.toml`.

La CLI no usa este módulo. La TUI carga el archivo al arrancar y la pantalla de
Configuración lo guarda de forma atómica. El formato es TOML y solo admite las
tres rutas documentadas.
"""

from __future__ import annotations

import contextlib
import json
import os
import tomllib
from dataclasses import dataclass, field
from pathlib import Path
from uuid import uuid4

from kamino.defaults import catalog_dir as default_catalog_dir
from kamino.defaults import snapshots_dir as default_snapshots_dir
from kamino.defaults import vault_dir as default_vault_dir
from kamino.errors import ConfigError

CATALOG_KEY = "catalog.dir"
CLONE_KEY = "clone.output_dir"
SNAPSHOTS_KEY = "deploy.snapshots_dir"

_SECTION_KEYS = {
    "catalog": {"dir": CATALOG_KEY},
    "clone": {"output_dir": CLONE_KEY},
    "deploy": {"snapshots_dir": SNAPSHOTS_KEY},
}
_TEMP_PREFIX = ".config-"
_CONFIG_DIR = ".config"
_CONFIG_PACKAGE = "kamino"
_CONFIG_NAME = "config.toml"


@dataclass(frozen=True)
class AppConfig:
    """Valores efectivos de las tres rutas de la TUI."""

    catalog_dir: Path
    clone_output_dir: Path
    snapshots_dir: Path


@dataclass(frozen=True)
class ConfigLoadResult:
    """Resultado de cargar la configuración, con valores parciales si es inválida."""

    config: AppConfig
    error: ConfigError | None = None
    exists: bool = False
    raw: dict[str, str] = field(default_factory=dict)


def config_path(home: Path | None = None) -> Path:
    """Ruta fija del archivo de configuración de la TUI."""
    return _home(home) / _CONFIG_DIR / _CONFIG_PACKAGE / _CONFIG_NAME


def default_config(home: Path | None = None) -> AppConfig:
    """Valores integrados de las tres rutas."""
    root = _home(home)
    return AppConfig(
        catalog_dir=default_catalog_dir(root),
        clone_output_dir=default_vault_dir(root),
        snapshots_dir=default_snapshots_dir(root),
    )


def load_config(*, home: Path | None = None, path: Path | None = None) -> ConfigLoadResult:
    """Carga la configuración de la TUI, conservando valores legibles si falla.

    Un archivo ausente no es un error. Un archivo inválido devuelve los defaults
    y los valores legibles en `raw` para que la pantalla de Configuración pueda
    corregirlos sin sobrescribir nada.
    """
    root = _home(home)
    target = path if path is not None else config_path(root)
    defaults = default_config(root)
    if not target.exists():
        return ConfigLoadResult(config=defaults)

    try:
        text = target.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError) as exc:
        return ConfigLoadResult(
            config=defaults,
            error=_invalid(target, f"no se pudo leer el archivo: {exc}"),
            exists=True,
        )

    try:
        data = tomllib.loads(text)
    except tomllib.TOMLDecodeError as exc:
        return ConfigLoadResult(
            config=defaults,
            error=_invalid(target, f"TOML inválido: {exc}"),
            exists=True,
        )

    values: dict[str, Path] = {}
    raw: dict[str, str] = {}
    errors: list[str] = []
    for section, section_data in data.items():
        allowed = _SECTION_KEYS.get(section)
        if allowed is None:
            errors.append(f"sección desconocida: [{section}]")
            continue
        if not isinstance(section_data, dict):
            errors.append(f"[{section}] debe ser una tabla")
            continue
        for key, value in section_data.items():
            full_key = f"{section}.{key}"
            if key not in allowed:
                errors.append(f"clave desconocida: {full_key}")
                continue
            if not isinstance(value, str):
                errors.append(f"{full_key} debe ser una cadena")
                continue
            raw[full_key] = value
            try:
                values[full_key] = resolve_path(value, home=root, key=full_key)
            except ConfigError as exc:
                errors.append(str(exc))

    config = AppConfig(
        catalog_dir=values.get(CATALOG_KEY, defaults.catalog_dir),
        clone_output_dir=values.get(CLONE_KEY, defaults.clone_output_dir),
        snapshots_dir=values.get(SNAPSHOTS_KEY, defaults.snapshots_dir),
    )
    if errors:
        return ConfigLoadResult(
            config=config,
            error=_invalid(target, "; ".join(errors)),
            exists=True,
            raw=raw,
        )
    return ConfigLoadResult(config=config, exists=True, raw=raw)


def resolve_path(value: str, *, home: Path | None = None, key: str = "ruta") -> Path:
    """Valida y expande una ruta configurada contra el home indicado."""
    root = _home(home)
    text = value.strip()
    if not text:
        raise ConfigError(f"{key} no puede estar vacía")
    if text == "~":
        path = root
    elif text.startswith("~/"):
        rest = text[2:]
        if rest.startswith("/"):
            raise ConfigError(f"{key}: solo se admite ~ o ~/ al inicio")
        path = root / rest
    elif text.startswith("~"):
        raise ConfigError(f"{key}: solo se admite ~ o ~/ al inicio")
    else:
        path = Path(text)
        if not path.is_absolute():
            raise ConfigError(f"{key}: usa una ruta absoluta o que empiece por ~")
    if path.exists() and not path.is_dir():
        raise ConfigError(f"{key}: la ruta existe y no es un directorio")
    return path


def save_config(config: AppConfig, *, home: Path | None = None, path: Path | None = None) -> Path:
    """Escribe las tres rutas de forma atómica y devuelve la ruta del archivo."""
    root = _home(home)
    target = path if path is not None else config_path(root)
    _validate_config_paths(config)
    content = _serialize(config)
    temp = target.with_name(f"{_TEMP_PREFIX}{uuid4().hex}.tmp")
    try:
        target.parent.mkdir(parents=True, exist_ok=True)
        fd = os.open(
            temp,
            os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0),
            0o644,
        )
        with os.fdopen(fd, "w", encoding="utf-8") as handle:
            handle.write(content)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temp, target)
    except OSError as exc:
        with contextlib.suppress(FileNotFoundError):
            temp.unlink()
        raise ConfigError(f"no se pudo guardar {target}: {exc}") from exc
    return target


def _validate_config_paths(config: AppConfig) -> None:
    values = (
        (CATALOG_KEY, config.catalog_dir),
        (CLONE_KEY, config.clone_output_dir),
        (SNAPSHOTS_KEY, config.snapshots_dir),
    )
    for key, path in values:
        if not path.is_absolute():
            raise ConfigError(f"{key}: usa una ruta absoluta o que empiece por ~")
        if path.exists() and not path.is_dir():
            raise ConfigError(f"{key}: la ruta existe y no es un directorio")


def _serialize(config: AppConfig) -> str:
    return (
        "[catalog]\n"
        f"dir = {_toml_string(config.catalog_dir)}\n"
        "\n"
        "[clone]\n"
        f"output_dir = {_toml_string(config.clone_output_dir)}\n"
        "\n"
        "[deploy]\n"
        f"snapshots_dir = {_toml_string(config.snapshots_dir)}\n"
    )


def _toml_string(path: Path) -> str:
    # JSON escaping is compatible with TOML basic strings for these values.
    return json.dumps(str(path), ensure_ascii=False)


def _invalid(target: Path, detail: str) -> ConfigError:
    return ConfigError(f"configuración inválida en {target}: {detail}")


def _home(home: Path | None) -> Path:
    return home if home is not None else Path.home()
