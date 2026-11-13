package tui

import (
	"github.com/charmbracelet/bubbles/key"

	"github.com/Gunmer/kamino/internal/texts"
)

func bind(keys []string, label, description string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(label, description))
}

func helpKey() key.Binding {
	return bind([]string{"?"}, "?", texts.KeyHelp)
}

func globalKeys() []key.Binding {
	return []key.Binding{
		helpKey(),
		bind([]string{"f1"}, "F1", texts.KeyHelp),
	}
}

func menuKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{"enter"}, "Enter", texts.KeyAccept),
	}
}

func cloneAppsKeys(filtering bool) []key.Binding {
	if filtering {
		return []key.Binding{
			bind([]string{"enter"}, "Enter", "aplicar"),
			bind([]string{"esc"}, "Esc", "cerrar"),
		}
	}
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{" "}, "Espacio", texts.KeyMark),
		bind([]string{"a", "n"}, "a/n", texts.KeyAllNone),
		bind([]string{"/"}, "/", texts.KeyFilter),
		bind([]string{"enter"}, "Enter", texts.KeyContinue),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func cloneDestinationKeys(option int, focus int, dialog bool) []key.Binding {
	if dialog {
		return []key.Binding{
			bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
			bind([]string{"enter"}, "Enter", texts.KeyAccept),
			bind([]string{"esc"}, "Esc", texts.KeyBack),
		}
	}
	keys := []key.Binding{bind([]string{"up", "k"}, "↑↓", texts.KeyMove)}
	if option == 2 || focus > 0 {
		keys = append(keys, bind([]string{"tab", "shift+tab"}, "Tab", texts.KeyField))
	}
	keys = append(keys,
		bind([]string{"enter"}, "Enter", texts.KeyChoose),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	)
	return keys
}

func cloneReviewKeys(phase clonePhase, hasSkipped, canCopy bool) []key.Binding {
	switch phase {
	case clonePhaseWriting:
		return []key.Binding{bind([]string{"ctrl+c"}, "Ctrl+C", texts.KeyCancel)}
	case clonePhaseDone:
		keys := []key.Binding{}
		if hasSkipped {
			keys = append(keys, bind([]string{"d"}, "d", texts.KeySkipped))
		}
		if canCopy {
			keys = append(keys, bind([]string{"c"}, "c", texts.KeyCopyPath))
		}
		return append(keys, bind([]string{"enter", "esc"}, "Enter/Esc", texts.KeyContinue))
	default:
		keys := []key.Binding{
			bind([]string{"d"}, "d", texts.KeySkipped),
			bind([]string{"f"}, "f", texts.KeyFiles),
			bind([]string{"enter"}, "Enter", texts.KeyClone),
			bind([]string{"esc"}, "Esc", texts.KeyBack),
			bind([]string{"ctrl+c"}, "Ctrl+C", texts.KeyCancel),
		}
		return keys
	}
}

func deployFileKeys(manual, filtering bool) []key.Binding {
	if manual || filtering {
		return []key.Binding{
			bind([]string{"enter"}, "Enter", "abrir"),
			bind([]string{"esc"}, "Esc", texts.KeyBack),
		}
	}
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{"/"}, "/", texts.KeyFilter),
		bind([]string{"o"}, "o", texts.KeyOtherFile),
		bind([]string{"enter"}, "Enter", texts.KeyContinue),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func deployAppsKeys(preparing bool) []key.Binding {
	if preparing {
		return []key.Binding{
			bind([]string{"esc", "ctrl+c"}, "Esc/Ctrl+C", texts.KeyCancel),
		}
	}
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{" "}, "Espacio", texts.KeyMark),
		bind([]string{"a", "n"}, "a/n", texts.KeyAllNone),
		bind([]string{"enter"}, "Enter", texts.KeyContinue),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func deployConflictsKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{"r"}, "r", texts.KeyReview),
		bind([]string{"c", "o"}, "c/o", texts.KeyCloneOriginal),
		bind([]string{"C", "O"}, "C/O", texts.KeyAll),
		bind([]string{"enter"}, "Enter", texts.KeyConfirm),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func deployDetailKeys(whole bool) []key.Binding {
	if whole {
		return []key.Binding{
			bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
			bind([]string{"enter"}, "Enter", texts.KeyAccept),
			bind([]string{"esc"}, "Esc", texts.KeyBack),
		}
	}
	return []key.Binding{
		bind([]string{"up", "k", "down", "j"}, "↑↓", texts.KeyLine),
		bind([]string{"left", "right", " "}, "←→/Espacio", texts.KeyChooseVersion),
		bind([]string{"c", "o"}, "c/o", texts.KeyBlock),
		bind([]string{"C", "O"}, "C/O", "archivo"),
		bind([]string{"tab", "shift+tab"}, "Tab", texts.KeyBlock),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func deployConfirmKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"left", "right"}, "←→", texts.KeyMove),
		bind([]string{"enter"}, "Enter", texts.KeyAccept),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func deploySummaryKeys(hasSnapshot bool) []key.Binding {
	keys := []key.Binding{}
	if hasSnapshot {
		keys = append(keys, bind([]string{"c"}, "c", texts.KeyCopySnapshot))
	}
	return append(keys, bind([]string{"enter", "esc"}, "Enter/Esc", texts.KeyContinue))
}

func updateKeys(phase updatePhase) []key.Binding {
	switch phase {
	case updatePhaseDownloading, updatePhaseVerifying, updatePhaseReplacing:
		return []key.Binding{bind([]string{"esc", "ctrl+c"}, "Esc/Ctrl+C", texts.KeyCancel)}
	default:
		return []key.Binding{
			bind([]string{"enter"}, "Enter", texts.KeyContinue),
			bind([]string{"esc"}, "Esc", "cancelar/salir"),
		}
	}
}

func settingsKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"tab", "shift+tab"}, "Tab/Mayús+Tab", texts.KeyMove),
		bind([]string{"ctrl+s"}, "Ctrl+S", texts.KeySave),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func helpKeys() []key.Binding {
	return []key.Binding{bind([]string{"enter", "esc", "?"}, "Enter/Esc/?", texts.KeyClose)}
}

func errorKeys(menuMode bool) []key.Binding {
	keys := []key.Binding{
		bind([]string{"up", "down", "left", "right"}, "↑↓", texts.KeyMove),
		bind([]string{"t"}, "t", texts.KeyDetail),
		bind([]string{"c"}, "c", texts.KeyCopy),
		bind([]string{"enter"}, "Enter", texts.KeyAccept),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
	if !menuMode {
		return keys[:4]
	}
	return keys
}
