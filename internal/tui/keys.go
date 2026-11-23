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

// fieldHelpKey is the shared help binding of a focused field. Using the same
// "?" key stops the footer from appending a duplicated "? ayuda" suffix.
func fieldHelpKey() key.Binding {
	return bind([]string{"f1", "?"}, "F1/?", texts.KeyHelp)
}

// fieldCursorKey is the shared cursor binding of a focused field.
func fieldCursorKey() key.Binding {
	return bind([]string{"left", "right"}, "←→", texts.KeyCursor)
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

func cloneAppsKeys(filtering bool, valueLength int) []key.Binding {
	if filtering {
		keys := []key.Binding{
			bind([]string{"enter"}, "Enter", texts.KeyApply),
			bind([]string{"esc"}, "Esc", texts.KeyClose),
		}
		if valueLength > 1 {
			keys = append(keys, fieldCursorKey())
		}
		return append(keys, fieldHelpKey())
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

func cloneDestinationKeys(option int, focus int, dialog bool, valueLength int) []key.Binding {
	if dialog {
		return []key.Binding{
			bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
			bind([]string{"enter"}, "Enter", texts.KeyAccept),
			bind([]string{"esc"}, "Esc", texts.KeyBack),
		}
	}
	if focus == 1 || focus == 2 {
		keys := []key.Binding{}
		if focus == 2 {
			keys = append(keys, bind([]string{"enter"}, "Enter", texts.KeySet))
		} else {
			keys = append(keys, bind([]string{"enter"}, "Enter", texts.KeyContinue))
		}
		keys = append(keys, bind([]string{"esc"}, "Esc", texts.KeyBack))
		if valueLength > 1 {
			keys = append(keys, fieldCursorKey())
		}
		if option == 2 {
			keys = append(keys, bind([]string{"tab", "shift+tab"}, "Tab", texts.KeyField))
		}
		return append(keys, fieldHelpKey())
	}
	keys := []key.Binding{bind([]string{"up", "k"}, "↑↓", texts.KeyMove)}
	if option == 2 {
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

func implantFileKeys(manual, filtering bool, valueLength int) []key.Binding {
	if manual || filtering {
		keys := []key.Binding{}
		if manual {
			keys = append(keys, bind([]string{"enter"}, "Enter", texts.KeyOpen))
		} else {
			keys = append(keys, bind([]string{"enter"}, "Enter", texts.KeyApply))
		}
		keys = append(keys, bind([]string{"esc"}, "Esc", texts.KeyBack))
		if valueLength > 1 {
			keys = append(keys, fieldCursorKey())
		}
		return append(keys, fieldHelpKey())
	}
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{"/"}, "/", texts.KeyFilter),
		bind([]string{"o"}, "o", texts.KeyOtherFile),
		bind([]string{"enter"}, "Enter", texts.KeyContinue),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func implantAppsKeys(preparing bool) []key.Binding {
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

func implantConflictsKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"up", "k"}, "↑↓", texts.KeyMove),
		bind([]string{"r"}, "r", texts.KeyReview),
		bind([]string{"c", "o"}, "c/o", texts.KeyCloneOriginal),
		bind([]string{"C", "O"}, "C/O", texts.KeyAll),
		bind([]string{"enter"}, "Enter", texts.KeyConfirm),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func implantDetailKeys(whole bool) []key.Binding {
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

func implantConfirmKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"left", "right"}, "←→", texts.KeyMove),
		bind([]string{"enter"}, "Enter", texts.KeyAccept),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
}

func implantSummaryKeys(hasSnapshot bool) []key.Binding {
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

func settingsKeys(valueLength int) []key.Binding {
	keys := []key.Binding{
		bind([]string{"tab", "shift+tab"}, "Tab/Mayús+Tab", texts.KeyMove),
		bind([]string{"ctrl+s"}, "Ctrl+S", texts.KeySave),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
	}
	if valueLength > 1 {
		keys = append(keys, fieldCursorKey())
	}
	return append(keys, fieldHelpKey())
}

func settingsAuthoringKeys() []key.Binding {
	return []key.Binding{
		bind([]string{"tab", "shift+tab"}, "Tab/Mayús+Tab", texts.KeyMove),
		bind([]string{"enter"}, "Enter", texts.KeyOpen),
		bind([]string{"ctrl+s"}, "Ctrl+S", texts.KeySave),
		bind([]string{"esc"}, "Esc", texts.KeyBack),
		fieldHelpKey(),
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
