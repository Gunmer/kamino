package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Gunmer/kamino/internal/conflicts"
)

func largeConflict(rowCount int) (conflicts.Conflict, []conflicts.RowDecision) {
	localLines := make([]string, 0, rowCount)
	cloneLines := make([]string, 0, rowCount)
	rows := make([]conflicts.HunkRow, 0, rowCount)
	for index := 0; index < rowCount; index++ {
		line := fmt.Sprintf("line-%05d", index)
		localLines = append(localLines, line)
		cloneLines = append(cloneLines, line+"-clone")
		rows = append(rows, conflicts.HunkRow{
			Kind: conflicts.RowReplace, Local: line + "\n", Clone: line + "-clone\n",
		})
	}
	local := strings.Join(localLines, "\n") + "\n"
	cloneText := strings.Join(cloneLines, "\n") + "\n"
	decisions := make([]conflicts.RowDecision, rowCount)
	for index := range decisions {
		decisions[index] = conflicts.RowClone
	}
	return conflicts.Conflict{
		AppID: "test", Path: ".config/large", Destination: "/Users/test/.config/large",
		Kind: conflicts.KindText, LocalText: &local, CloneText: &cloneText,
		Hunks: []conflicts.Hunk{{
			Index: 1, LocalStart: 1, LocalEnd: rowCount + 1,
			CloneStart: 1, CloneEnd: rowCount + 1, Rows: rows,
		}},
	}, decisions
}

func BenchmarkD3LargeFile(b *testing.B) {
	conflict, decisions := largeConflict(20000)
	app := testApp(false, true)
	app.conflicts = []*conflictState{{Conflict: conflict, Action: conflicts.ActionReview, Rows: decisions}}
	screen := newDeployDetailScreen(app)
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		_ = screen.View(96, 35)
	}
}
