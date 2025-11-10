package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

var (
	// Global UI state
	quiet        bool
	verbose      bool
	jsonOutput   bool
	colorEnabled bool

	// Output streams for testing
	outputStream      io.Writer = os.Stdout
	errorOutputStream io.Writer = os.Stderr
)

// InitializeFromEnv initializes UI settings from environment variables
func InitializeFromEnv() {
	// Check if we're in a terminal
	colorEnabled = isatty.IsTerminal(os.Stdout.Fd()) && os.Getenv("NO_COLOR") == ""

	// Check environment variables
	if os.Getenv("APX_QUIET") != "" {
		quiet = true
	}
	if os.Getenv("APX_VERBOSE") != "" {
		verbose = true
	}
	if os.Getenv("APX_JSON") != "" {
		jsonOutput = true
	}
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
	}
}

// SetQuiet enables or disables quiet mode
func SetQuiet(q bool) {
	quiet = q
}

// SetVerbose enables or disables verbose mode
func SetVerbose(v bool) {
	verbose = v
}

// SetJSONOutput enables or disables JSON output
func SetJSONOutput(j bool) {
	jsonOutput = j
}

// SetColorEnabled enables or disables colored output
func SetColorEnabled(c bool) {
	colorEnabled = c
	color.NoColor = !c
}

// SetOutput sets the output stream for normal messages
func SetOutput(w io.Writer) {
	if w == nil {
		outputStream = os.Stdout
	} else {
		outputStream = w
	}
}

// SetErrorOutput sets the output stream for error messages
func SetErrorOutput(w io.Writer) {
	if w == nil {
		errorOutputStream = os.Stderr
	} else {
		errorOutputStream = w
	}
}

// Message represents a structured message for JSON output
type Message struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Time    string `json:"time,omitempty"`
}

// print outputs a message with the given level and color
func print(level string, colorFunc func(string, ...interface{}) string, format string, args ...interface{}) {
	if quiet && level != "error" {
		return
	}

	message := fmt.Sprintf(format, args...)

	// Choose appropriate output stream
	output := outputStream
	if level == "error" {
		output = errorOutputStream
	}

	if jsonOutput {
		msg := Message{
			Level:   level,
			Message: message,
		}
		if data, err := json.Marshal(msg); err == nil {
			fmt.Fprintln(output, string(data))
		}
		return
	}

	if colorEnabled && colorFunc != nil {
		fmt.Fprintln(output, colorFunc(message))
	} else {
		fmt.Fprintln(output, message)
	}
}

// Info prints an informational message
func Info(format string, args ...interface{}) {
	print("info", color.New(color.FgCyan).SprintfFunc(), format, args...)
}

// Success prints a success message
func Success(format string, args ...interface{}) {
	print("success", color.New(color.FgGreen).SprintfFunc(), format, args...)
}

// Warning prints a warning message
func Warning(format string, args ...interface{}) {
	print("warning", color.New(color.FgYellow).SprintfFunc(), format, args...)
}

// Error prints an error message
func Error(format string, args ...interface{}) {
	print("error", color.New(color.FgRed).SprintfFunc(), format, args...)
}

// Debug prints a debug message (only in verbose mode)
func Debug(format string, args ...interface{}) {
	if !verbose {
		return
	}
	print("debug", color.New(color.FgMagenta).SprintfFunc(), "[DEBUG] "+format, args...)
}

// Verbose prints a verbose message (only in verbose mode)
func Verbose(format string, args ...interface{}) {
	if !verbose {
		return
	}
	print("verbose", nil, format, args...)
}

// Progress represents a progress indicator
type Progress struct {
	current int
	total   int
	message string
}

// NewProgress creates a new progress indicator
func NewProgress(total int, message string) *Progress {
	return &Progress{
		current: 0,
		total:   total,
		message: message,
	}
}

// Update updates the progress
func (p *Progress) Update(current int, message string) {
	p.current = current
	if message != "" {
		p.message = message
	}

	if jsonOutput {
		msg := map[string]interface{}{
			"level":   "progress",
			"current": p.current,
			"total":   p.total,
			"message": p.message,
		}
		if data, err := json.Marshal(msg); err == nil {
			fmt.Println(string(data))
		}
		return
	}

	if !quiet {
		percentage := float64(p.current) / float64(p.total) * 100
		fmt.Printf("\r%s [%d/%d] %.1f%%", p.message, p.current, p.total, percentage)
		if p.current == p.total {
			fmt.Println() // New line when complete
		}
	}
}

// Finish marks the progress as complete
func (p *Progress) Finish() {
	p.Update(p.total, p.message)
}

// Confirm asks for user confirmation
func Confirm(message string) bool {
	if jsonOutput {
		// In JSON mode, assume yes for automation
		return true
	}

	fmt.Printf("%s [y/N]: ", message)
	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// Table represents a simple table for output
type Table struct {
	Headers []string
	Rows    [][]string
}

// NewTable creates a new table
func NewTable(headers ...string) *Table {
	return &Table{
		Headers: headers,
		Rows:    make([][]string, 0),
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(row ...string) {
	t.Rows = append(t.Rows, row)
}

// Print prints the table
func (t *Table) Print() {
	if jsonOutput {
		data := map[string]interface{}{
			"level":   "table",
			"headers": t.Headers,
			"rows":    t.Rows,
		}
		if jsonData, err := json.Marshal(data); err == nil {
			fmt.Println(string(jsonData))
		}
		return
	}

	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(t.Headers))
	for i, header := range t.Headers {
		widths[i] = len(header)
	}

	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print headers
	if len(t.Headers) > 0 {
		for i, header := range t.Headers {
			fmt.Printf("%-*s", widths[i]+2, header)
		}
		fmt.Println()

		// Print separator
		for i := range t.Headers {
			fmt.Printf("%s", strings.Repeat("-", widths[i]+2))
		}
		fmt.Println()
	}

	// Print rows
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%-*s", widths[i]+2, cell)
			}
		}
		fmt.Println()
	}
}
