"""Salida no interactiva: avisos, resúmenes y diffs con Rich."""

from __future__ import annotations

import difflib

from rich.console import Console
from rich.table import Table
from rich.text import Text

from kamino import texts
from kamino.update import UpdateOutcome

_console = Console()


def print_privacy_warning() -> None:
    """Muestra el aviso de contenido sensible antes de clonar."""
    _console.print(f"[yellow]{texts.PRIVACY_WARNING}[/yellow]")


def print_update_start() -> None:
    """Muestra que comienza la comprobación de la última versión."""
    _console.print(texts.UPDATE_CHECKING)


def print_update_result(result) -> None:
    """Imprime el resultado de la actualización o de la consulta."""
    if result.outcome is UpdateOutcome.UPDATED:
        _console.print(
            texts.UPDATE_UPDATED.format(
                current=result.current_version,
                latest=result.latest_version,
                path=result.target,
            )
        )
        _console.print(texts.UPDATE_RESTART)
        return
    if result.outcome is UpdateOutcome.NEWER:
        _console.print(
            texts.UPDATE_NEWER.format(current=result.current_version, latest=result.latest_version)
        )
        return
    _console.print(texts.UPDATE_UP_TO_DATE.format(current=result.current_version))


def print_clone_summary(result, *, verbose: bool = False) -> None:
    """Imprime el resumen de la clonación, o el plan en modo dry-run."""
    if result.dry_run:
        _console.print(f"[bold]{texts.CLONE_DRY_RUN_TITLE}[/bold]")
        for operation in result.plan.operations:
            _console.print(f"  + {operation.path}")
        for skipped in result.plan.skipped:
            _console.print(f"  ~ {skipped.path} ({skipped.reason})")
        for notice in result.plan.notices:
            _console.print(f"  ! {notice}")
        if result.plan_output is not None:
            _console.print(texts.CLONE_DESTINATION.format(path=result.plan_output))
        return
    table = Table(title=texts.CLONE_SUMMARY_TITLE)
    table.add_column(texts.CLONE_APPS_LABEL, style="bold")
    table.add_column(texts.CLONE_FILES_LABEL, justify="right")
    for app in result.manifest.apps:
        table.add_row(app.name, str(len(app.files)))
    _console.print(table)
    _console.print(texts.CLONE_DESTINATION.format(path=result.output))
    if verbose:
        _console.print(f"[bold]{texts.CLONE_INCLUDED_TITLE}[/bold]")
        for operation in result.plan.operations:
            _console.print(f"  + {operation.path}")
        for skipped in result.plan.skipped:
            _console.print(f"  ~ {skipped.path} ({skipped.reason})")
        for notice in result.plan.notices:
            _console.print(f"  ! {notice}")


def print_deploy_summary(result, *, verbose: bool = False) -> None:
    """Print the deploy summary, or the plan and diffs in dry-run mode."""
    if result.dry_run:
        _console.print(f"[bold]{texts.TITLE_DEPLOY_DRY_RUN}[/bold]")
        for entry in result.plan.entries:
            _console.print(f"  [{entry.status}] {entry.path}")
        for conflict in result.plan.conflicts:
            _print_conflict_diff(conflict)
        _console.print(texts.DEPLOY_OMITTED.format(count=len(result.omitted)))
        _console.print(texts.DEPLOY_UNCHANGED.format(count=len(result.unchanged)))
        _console.print(texts.NOT_INSTALLED + f": {len(result.not_installed)}")
        for notice in result.plan.notices:
            _console.print(f"  ! {notice}")
        return
    _console.print(texts.DEPLOY_DEPLOYED.format(count=len(result.deployed)))
    _console.print(texts.DEPLOY_HYBRIDS.format(count=len(result.merged)))
    _console.print(texts.DEPLOY_OMITTED.format(count=len(result.omitted)))
    _console.print(texts.DEPLOY_UNCHANGED.format(count=len(result.unchanged)))
    _console.print(texts.NOT_INSTALLED + f": {len(result.not_installed)}")
    if result.snapshot is not None:
        _console.print(texts.DEPLOY_SNAPSHOT_PATH.format(path=result.snapshot))
        _console.print(texts.DEPLOY_UNDO.format(path=result.snapshot))
    else:
        _console.print(texts.DEPLOY_NO_SNAPSHOT)
    for notice in result.plan.notices:
        _console.print(f"[yellow]! {notice}[/yellow]")
    if verbose:
        _console.print(f"[bold]{texts.TITLE_DETAIL}[/bold]")
        for path in result.deployed:
            _console.print(f"  + {path}")
        for path in result.merged:
            _console.print(f"  * {path}")
        for path in result.omitted:
            _console.print(f"  ~ {path}")
        for path in result.unchanged:
            _console.print(f"  = {path}")


def _print_conflict_diff(conflict) -> None:
    """Write one conflict diff or a whole-file note to stdout."""
    if conflict.kind.value == "text":
        if conflict.local_text is None or conflict.clone_text is None:
            return
        diff = difflib.unified_diff(
            conflict.local_text.splitlines(),
            conflict.clone_text.splitlines(),
            fromfile=f"{texts.ORIGINAL}/{conflict.path}",
            tofile=f"clon/{conflict.path}",
            lineterm="",
        )
        _console.print(Text("\n".join(diff)), markup=False, highlight=False)
        return
    if conflict.kind.value == "symlink":
        _console.print(
            "  ! "
            + texts.CONFLICT_SYMLINK.format(
                path=conflict.path,
                original=texts.ORIGINAL,
                local=conflict.local_link,
                clone=conflict.clone_link,
            )
        )
        return
    reason = "binario" if conflict.kind.value == "binary" else "grande (>5 MB)"
    _console.print(f"  ! {conflict.path}: conflicto {reason}; se resuelve como archivo entero")
