// Package output renders human-readable tables and stable JSON.
//
// Human output is concise; machine output is stable and free of styling.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Printer writes results to stdout and diagnostics to stderr.
type Printer struct {
	out    io.Writer
	err    io.Writer
	asJSON bool
	color  bool
}

// New builds a Printer.
func New(out, errOut io.Writer, asJSON, color bool) *Printer {
	return &Printer{out: out, err: errOut, asJSON: asJSON, color: color}
}

// JSON reports whether JSON output is enabled.
func (p *Printer) JSON() bool { return p.asJSON }

// JSONValue writes v as indented JSON followed by a newline.
func (p *Printer) JSONValue(v any) error {
	encoder := json.NewEncoder(p.out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(v)
}

// Line writes a formatted line to stdout.
func (p *Printer) Line(format string, args ...any) {
	fmt.Fprintf(p.out, format+"\n", args...)
}

// Field writes an aligned "Label  value" line to stdout.
func (p *Printer) Field(label, value string) {
	fmt.Fprintf(p.out, "%-13s %s\n", label, value)
}

// Errorf writes a diagnostic line to stderr.
func (p *Printer) Errorf(format string, args ...any) {
	fmt.Fprintf(p.err, format+"\n", args...)
}

// Table writes a left-aligned table with a header row.
func (p *Printer) Table(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = utf8.RuneCountInString(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && utf8.RuneCountInString(cell) > widths[i] {
				widths[i] = utf8.RuneCountInString(cell)
			}
		}
	}

	p.writeRow(headers, widths, true)
	for _, row := range rows {
		p.writeRow(row, widths, false)
	}
}

func (p *Printer) writeRow(row []string, widths []int, header bool) {
	var builder strings.Builder
	for i, width := range widths {
		cell := ""
		if i < len(row) {
			cell = row[i]
		}
		if i == len(widths)-1 {
			builder.WriteString(cell)
			continue
		}
		builder.WriteString(cell)
		builder.WriteString(strings.Repeat(" ", width-utf8.RuneCountInString(cell)))
		builder.WriteString("  ")
	}

	line := strings.TrimRight(builder.String(), " ")
	if header && p.color {
		line = "\x1b[1m" + line + "\x1b[0m"
	}
	fmt.Fprintln(p.out, line)
}

// SupportsColor reports whether ANSI styling should be used for w. Styling is
// disabled when requested, when NO_COLOR is set, when TERM=dumb, or when w is
// not a terminal.
func SupportsColor(w io.Writer, noColor bool, lookup func(string) (string, bool)) bool {
	if noColor {
		return false
	}
	if value, ok := lookup("NO_COLOR"); ok && value != "" {
		return false
	}
	if value, ok := lookup("TERM"); ok && value == "dumb" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
