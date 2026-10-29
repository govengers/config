package config

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

const (
	maskedValue          = "****"
	maximumDisplayLength = 50
)

// Print writes a table of all loaded values to stdout. Sensitive values are masked.
func Print() {
	Fprint(os.Stdout)
}

// Fprint writes the table Print writes to writer.
func Fprint(writer io.Writer) {
	tableWriter := table.NewWriter()
	tableWriter.SetTitle("Application Configuration")
	tableWriter.SetOutputMirror(writer)
	tableWriter.AppendHeader(table.Row{"ENV VAR", "Type", "Default Value", "Value Provided", "Current Value"})
	tableWriter.AppendRows(registry.rows())
	tableWriter.SetStyle(table.StyleColoredYellowWhiteOnBlack)
	tableWriter.Render()
}

func (r *Registry) rows() []table.Row {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	rows := make([]table.Row, 0, len(r.entries))
	for _, key := range slices.Sorted(maps.Keys(r.entries)) {
		loaded := r.entries[key]
		rows = append(rows, table.Row{
			key,
			string(loaded.declaration.valueType),
			displayDefault(loaded.declaration),
			loaded.provided,
			display(loaded.declaration, loaded.value),
		})
	}
	return rows
}

func displayDefault(current declaration) string {
	if current.defaultValue == nil {
		return "-"
	}
	return display(current, current.defaultValue)
}

// display formats a value for the table. A sensitive value is masked with a fixed width, so the
// table does not reveal its length either.
func display(current declaration, value any) string {
	if current.sensitive {
		return maskedValue
	}
	return truncate(format(value))
}

func format(value any) string {
	if items, isList := value.([]string); isList {
		return strings.Join(items, ", ")
	}
	return fmt.Sprint(value)
}

func truncate(text string) string {
	characters := []rune(text)
	if len(characters) <= maximumDisplayLength {
		return text
	}
	return string(characters[:maximumDisplayLength-3]) + "..."
}
