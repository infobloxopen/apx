package testhelpers

import (
	"bytes"
	"os"
	"sync"

	"github.com/infobloxopen/apx/internal/ui"
)

// TestOutput provides test output capture functionality that captures both os.Stdout/Stderr and UI package output
type TestOutput struct {
	stdout       bytes.Buffer
	stderr       bytes.Buffer
	oldStdout    *os.File
	oldStderr    *os.File
	stdoutReader *os.File
	stderrReader *os.File
	stdoutWriter *os.File
	stderrWriter *os.File
	mu           sync.Mutex
}

// NewTestOutput creates a new test output capturer
func NewTestOutput() *TestOutput {
	return &TestOutput{}
}

// Setup starts capturing output by redirecting both os.Stdout/Stderr and UI package streams
func (t *TestOutput) Setup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Reset buffers
	t.stdout.Reset()
	t.stderr.Reset()

	// Save original outputs
	t.oldStdout = os.Stdout
	t.oldStderr = os.Stderr

	// Create pipes
	var err error
	t.stdoutReader, t.stdoutWriter, err = os.Pipe()
	if err != nil {
		panic(err)
	}

	t.stderrReader, t.stderrWriter, err = os.Pipe()
	if err != nil {
		panic(err)
	}

	// Redirect os.Stdout and os.Stderr
	os.Stdout = t.stdoutWriter
	os.Stderr = t.stderrWriter

	// Also set UI package to write to our writers (for any UI package output)
	ui.SetOutput(t.stdoutWriter)
	ui.SetErrorOutput(t.stderrWriter)
}

// Restore stops capturing and restores original output
func (t *TestOutput) Restore() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Close writers to signal end of output
	if t.stdoutWriter != nil {
		t.stdoutWriter.Close()
		// Read all remaining data from stdout
		t.stdout.ReadFrom(t.stdoutReader)
		t.stdoutReader.Close()
	}

	if t.stderrWriter != nil {
		t.stderrWriter.Close()
		// Read all remaining data from stderr
		t.stderr.ReadFrom(t.stderrReader)
		t.stderrReader.Close()
	}

	// Restore original outputs
	if t.oldStdout != nil {
		os.Stdout = t.oldStdout
	}
	if t.oldStderr != nil {
		os.Stderr = t.oldStderr
	}

	// Reset UI package to default outputs
	ui.SetOutput(nil)
	ui.SetErrorOutput(nil)
} // StdoutString returns captured stdout as string
func (t *TestOutput) StdoutString() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stdout.String()
}

// StderrString returns captured stderr as string
func (t *TestOutput) StderrString() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stderr.String()
}

// Reset clears the captured output
func (t *TestOutput) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stdout.Reset()
	t.stderr.Reset()
}
