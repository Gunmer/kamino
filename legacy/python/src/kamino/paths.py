"""Normalización de rutas relativas al home y acceso seguro.

Todo se guarda relativo a `~` (diseño técnico §1 y §4.1), lo que resuelve la
migración entre usuarios. Las rutas fuera del home no se admiten en la v1.

Además de la validación léxica, `SafeHome` abre el home y recorre sus
descendientes con descriptores de directorio y `O_NOFOLLOW`. Así un enlace
intermedio no puede redirigir una lectura o escritura fuera del home, ni
siquiera si la jerarquía cambia entre la planificación y la ejecución.
"""

from __future__ import annotations

import os
import stat
from collections.abc import Callable, Iterator
from contextlib import contextmanager, suppress
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Literal
from uuid import uuid4

from kamino.errors import PathError

_DIRECTORY_FLAGS = os.O_RDONLY | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_NOFOLLOW", 0)
_FILE_FLAGS = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
_TEMP_PREFIX = ".kamino-"
_CHUNK = 65536


def to_absolute(path: str | Path, *, home: Path | None = None) -> Path:
    """Resuelve una ruta relativa del home a su forma absoluta.

    `home` por defecto es el home del usuario actual. Eleva `PathError` si la
    ruta está vacía, es absoluta, contiene `..` o empieza por `~`, porque
    cualquiera de esos casos saldría del home.
    """
    _ensure_relative(path)
    root = home if home is not None else Path.home()
    return root / str(path)


def to_relative(path: str | Path, *, home: Path | None = None) -> Path:
    """Convierte una ruta absoluta del home a su forma relativa.

    `home` por defecto es el home del usuario actual. Eleva `PathError` si la
    ruta no es absoluta o queda fuera del home (incluido el caso de un
    prefijo de texto como `/home/usuario2` frente a `/home/usuario`).
    """
    candidate = Path(os.path.normpath(str(path)))
    if not candidate.is_absolute():
        raise PathError(f"se requiere una ruta absoluta: {str(path)!r}")
    root = Path(os.path.normpath(str(home if home is not None else Path.home())))
    try:
        return candidate.relative_to(root)
    except ValueError as exc:
        raise PathError(f"ruta fuera del home {str(root)!r}: {str(path)!r}") from exc


def _ensure_relative(path: str | Path) -> None:
    """Rechaza rutas que escapan del home (misma semántica que el catálogo)."""
    text = str(path)
    if not text:
        raise PathError("ruta vacía no admitida")
    if text.startswith("~"):
        raise PathError(f"ruta con '~' no admitida, usa una ruta relativa al home: {text!r}")
    pure = PurePosixPath(text)
    if pure.is_absolute():
        raise PathError(f"ruta absoluta no admitida: {text!r}")
    if ".." in pure.parts:
        raise PathError(f"ruta fuera del home no admitida: {text!r}")


@dataclass(frozen=True)
class WalkItem:
    """Un elemento descubierto al recorrer un directorio sin seguir enlaces.

    `kind` distingue archivos regulares, enlaces simbólicos, tipos especiales
    y entradas que no se pudieron leer. `path` es siempre relativo al home.
    """

    path: str
    kind: Literal["file", "link", "special", "skipped"]
    info: os.stat_result | None = None


class SafeHome:
    """Acceso a rutas dentro de un home sin atravesar enlaces intermedios.

    El descriptor del home se abre una vez y todas las operaciones relativas
    usan `dir_fd`. Los componentes intermedios se abren con `O_NOFOLLOW`; el
    componente final se inspecciona con `lstat` para poder representar enlaces
    simbólicos sin seguirlos. Si la plataforma no ofrece estas primitivas, se
    aborta en lugar de degradar a una comprobación léxica.
    """

    def __init__(self, home: Path | None = None, *, create: bool = False):
        if not hasattr(os, "O_NOFOLLOW") or not hasattr(os, "O_DIRECTORY"):
            raise PathError("esta plataforma no permite confinar el acceso al home")
        if os.open not in os.supports_dir_fd:
            raise PathError("esta plataforma no permite abrir rutas relativas a un directorio")
        self.root = Path(home if home is not None else Path.home())
        if create:
            self.root.mkdir(parents=True, exist_ok=True)
        try:
            self._root_fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY)
        except OSError as exc:
            raise PathError(f"no se pudo abrir el home {str(self.root)!r}: {exc}") from exc

    def close(self) -> None:
        fd = getattr(self, "_root_fd", None)
        if fd is not None:
            os.close(fd)
            self._root_fd = None

    def __enter__(self) -> SafeHome:
        return self

    def __exit__(self, *_exc: object) -> None:
        self.close()

    @staticmethod
    def parts(rel: str | Path) -> tuple[str, ...]:
        """Valida una ruta relativa y devuelve sus componentes normalizados."""
        text = str(rel)
        _ensure_relative(text)
        return tuple(part for part in PurePosixPath(text).parts if part not in ("", "."))

    def _step_dir(self, parent_fd: int, part: str, rel: str | Path) -> int:
        """Abre un componente intermedio como directorio y cierra el padre."""
        try:
            child = os.open(part, _DIRECTORY_FLAGS, dir_fd=parent_fd)
        except FileNotFoundError:
            raise
        except OSError as exc:
            raise PathError(f"no se pudo acceder a {str(rel)!r} sin seguir enlaces: {exc}") from exc
        os.close(parent_fd)
        return child

    @contextmanager
    def _parent(self, rel: str | Path) -> Iterator[tuple[int, str]]:
        """Abre el directorio padre sin seguir enlaces y devuelve `(fd, nombre)`."""
        parts = self.parts(rel)
        if not parts:
            raise PathError("se requiere una ruta dentro del home, no el home")
        fd = os.dup(self._root_fd)
        try:
            for part in parts[:-1]:
                fd = self._step_dir(fd, part, rel)
            yield fd, parts[-1]
        finally:
            os.close(fd)

    @contextmanager
    def _open_dir(self, rel: str | Path) -> Iterator[int]:
        """Abre un directorio del home sin seguir enlaces intermedios ni finales."""
        parts = self.parts(rel)
        fd = os.dup(self._root_fd)
        try:
            for part in parts:
                fd = self._step_dir(fd, part, rel)
            yield fd
        finally:
            os.close(fd)

    def lstat(self, rel: str | Path) -> os.stat_result:
        """Devuelve `lstat` del componente final sin seguir enlaces intermedios."""
        with self._parent(rel) as (parent_fd, name):
            return os.stat(name, dir_fd=parent_fd, follow_symlinks=False)

    def read_bytes(self, rel: str | Path) -> bytes:
        """Lee un archivo regular sin seguir enlaces intermedios ni finales."""
        with self._parent(rel) as (parent_fd, name):
            try:
                fd = os.open(name, _FILE_FLAGS, dir_fd=parent_fd)
            except OSError as exc:
                raise PathError(f"no se pudo leer {str(rel)!r} sin seguir enlaces: {exc}") from exc
            try:
                info = os.fstat(fd)
                if not stat.S_ISREG(info.st_mode):
                    raise PathError(f"no es un archivo regular: {str(rel)!r}")
                chunks: list[bytes] = []
                while chunk := os.read(fd, _CHUNK):
                    chunks.append(chunk)
                return b"".join(chunks)
            finally:
                os.close(fd)

    def read_text(self, rel: str | Path, *, limit: int | None = None) -> str:
        """Lee un archivo como UTF-8, opcionalmente rechazando tamaños mayores a `limit`."""
        if limit is not None:
            info = self.lstat(rel)
            if info.st_size > limit:
                raise PathError(f"archivo demasiado grande para analizar: {str(rel)!r}")
        return self.read_bytes(rel).decode("utf-8", errors="replace")

    def readlink(self, rel: str | Path) -> str:
        """Devuelve el destino de un enlace final sin seguirlo."""
        with self._parent(rel) as (parent_fd, name):
            try:
                return os.readlink(name, dir_fd=parent_fd)
            except OSError as exc:
                raise PathError(f"no es un enlace simbólico: {str(rel)!r}: {exc}") from exc

    def listdir(self, rel: str | Path) -> list[str]:
        """Lista los nombres de un directorio del home, ordenados."""
        with self._open_dir(rel) as fd, os.scandir(fd) as entries:
            return sorted(entry.name for entry in entries)

    def walk(
        self,
        rel: str | Path,
        *,
        prune: Callable[[str], bool] | None = None,
    ) -> Iterator[WalkItem]:
        """Recorre un directorio sin seguir enlaces y en orden determinista.

        `prune` recibe la ruta relativa de cada entrada y devuelve `True` para
        omitirla por completo (por ejemplo, una exclusión de configuración).
        Los enlaces se devuelven como `link` y no se recorren.
        """
        yield from self._walk_dir(str(rel), prune or (lambda _path: False))

    def _walk_dir(self, rel: str, prune: Callable[[str], bool]) -> Iterator[WalkItem]:
        try:
            with self._open_dir(rel) as fd:
                with os.scandir(fd) as entries:
                    ordered = sorted(entries, key=lambda entry: entry.name)
                directories: list[str] = []
                for entry in ordered:
                    child_rel = (PurePosixPath(rel) / entry.name).as_posix()
                    if prune(child_rel):
                        continue
                    try:
                        if entry.is_symlink():
                            yield WalkItem(child_rel, "link", entry.stat(follow_symlinks=False))
                        elif entry.is_dir(follow_symlinks=False):
                            directories.append(entry.name)
                        elif entry.is_file(follow_symlinks=False):
                            yield WalkItem(child_rel, "file", entry.stat(follow_symlinks=False))
                        else:
                            yield WalkItem(child_rel, "special", entry.stat(follow_symlinks=False))
                    except OSError:
                        yield WalkItem(child_rel, "skipped")
                for name in directories:
                    child_rel = (PurePosixPath(rel) / name).as_posix()
                    yield from self._walk_dir(child_rel, prune)
        except FileNotFoundError:
            yield WalkItem(rel, "skipped")
        except PathError:
            yield WalkItem(rel, "skipped")

    def ensure_parent(self, rel: str | Path) -> tuple[int, str]:
        """Crea los directorios padre que falten y devuelve `(fd, nombre)`.

        El descriptor devuelto queda abierto y el llamador debe cerrarlo. Si un
        componente existente es un enlace o no es un directorio, se rechaza.
        """
        parts = self.parts(rel)
        if not parts:
            raise PathError("se requiere una ruta dentro del home, no el home")
        fd = os.dup(self._root_fd)
        try:
            for part in parts[:-1]:
                try:
                    child = os.open(part, _DIRECTORY_FLAGS, dir_fd=fd)
                except FileNotFoundError:
                    try:
                        os.mkdir(part, mode=0o755, dir_fd=fd)
                    except FileExistsError:
                        pass
                    except OSError as exc:
                        raise PathError(
                            f"no se pudo crear el directorio de {str(rel)!r}: {exc}"
                        ) from exc
                    try:
                        child = os.open(part, _DIRECTORY_FLAGS, dir_fd=fd)
                    except OSError as exc:
                        raise PathError(
                            f"no se pudo acceder a {str(rel)!r} sin seguir enlaces: {exc}"
                        ) from exc
                except OSError as exc:
                    raise PathError(
                        f"no se pudo acceder a {str(rel)!r} sin seguir enlaces: {exc}"
                    ) from exc
                os.close(fd)
                fd = child
        except Exception:
            os.close(fd)
            raise
        return fd, parts[-1]

    def write_file(self, rel: str | Path, data: bytes, *, mode: int = 0o644) -> None:
        """Escribe un archivo regular con reemplazo atómico y sin seguir enlaces."""
        parent_fd, name = self.ensure_parent(rel)
        temp = f"{_TEMP_PREFIX}{uuid4().hex}.tmp"
        try:
            fd = os.open(
                temp,
                os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0),
                mode,
                dir_fd=parent_fd,
            )
            try:
                with os.fdopen(fd, "wb") as handle:
                    handle.write(data)
                    handle.flush()
                    os.fchmod(handle.fileno(), mode)
                os.replace(temp, name, src_dir_fd=parent_fd, dst_dir_fd=parent_fd)
            except Exception:
                with suppress(FileNotFoundError):
                    os.unlink(temp, dir_fd=parent_fd)
                raise
        finally:
            os.close(parent_fd)

    def replace_with_symlink(self, rel: str | Path, target: str) -> None:
        """Crea un enlace simbólico con reemplazo atómico dentro del home."""
        parent_fd, name = self.ensure_parent(rel)
        temp = f"{_TEMP_PREFIX}{uuid4().hex}.tmp"
        try:
            try:
                os.symlink(target, temp, dir_fd=parent_fd)
                os.replace(temp, name, src_dir_fd=parent_fd, dst_dir_fd=parent_fd)
            except Exception:
                with suppress(FileNotFoundError):
                    os.unlink(temp, dir_fd=parent_fd)
                raise
        finally:
            os.close(parent_fd)
