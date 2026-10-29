package config

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/jedib0t/go-pretty/v6/table"
)

const (
	maskedValue          = "****"
	maximumDisplayLength = 50
)

// Print writes a table of all loaded values to stdout. Sensitive values are masked.
func Print() {
	_ = Fprint(os.Stdout) // a failed write to stdout has nowhere else to be reported
}

// Fprint writes the table Print writes to writer and returns the write error, if any.
func Fprint(writer io.Writer) error {
	tableWriter := table.NewWriter()
	tableWriter.SetTitle("Application Configuration")
	tableWriter.AppendHeader(table.Row{"ENV VAR", "Type", "Default Value", "Value Provided", "Current Value"})
	tableWriter.AppendRows(registry.rows())
	tableWriter.SetStyle(table.StyleColoredYellowWhiteOnBlack)
	_, err := io.WriteString(writer, tableWriter.Render()+"\n")
	return err
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

// format renders a value for the table and drops control characters, so a value cannot inject
// terminal escape sequences or extra lines into logs.
func format(value any) string {
	text := fmt.Sprint(value)
	if items, isList := value.([]string); isList {
		text = strings.Join(items, ", ")
	}
	return strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return -1
		}
		return character
	}, text)
}

func truncate(text string) string {
	characters := []rune(text)
	if len(characters) <= maximumDisplayLength {
		return text
	}
	return string(characters[:maximumDisplayLength-3]) + "..."
}
