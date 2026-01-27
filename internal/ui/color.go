// Package ui provides terminal output formatting with color support.
package ui

import (
	"fmt"
	"io"
	"os"
)

// ANSI color codes
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

// Printer handles colored terminal output
type Printer struct {
	out     io.Writer
	noColor bool
}

// New creates a new Printer that writes to the given output.
// Color is disabled if the NO_COLOR environment variable is set.
func New(out io.Writer) *Printer {
	_, noColor := os.LookupEnv("NO_COLOR")
	return &Printer{
		out:     out,
		noColor: noColor,
	}
}

// Default creates a Printer that writes to stdout.
func Default() *Printer {
	return New(os.Stdout)
}

// Header prints a cyan bold message (for section headers).
func (p *Printer) Header(format string, args ...any) {
	p.print(cyan+bold, format, args...)
}

// Success prints a green message.
func (p *Printer) Success(format string, args ...any) {
	p.print(green, format, args...)
}

// Warning prints a yellow message.
func (p *Printer) Warning(format string, args ...any) {
	p.print(yellow, format, args...)
}

// Error prints a red message.
func (p *Printer) Error(format string, args ...any) {
	p.print(red, format, args...)
}

// Info prints a message with no color.
func (p *Printer) Info(format string, args ...any) {
	fmt.Fprintf(p.out, format+"\n", args...)
}

func (p *Printer) print(color, format string, args ...any) {
	if p.noColor {
		fmt.Fprintf(p.out, format+"\n", args...)
		return
	}
	fmt.Fprintf(p.out, color+format+reset+"\n", args...)
}
