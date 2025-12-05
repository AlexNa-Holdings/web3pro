package command

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func NewErrorsCommand() *Command {
	return &Command{
		Command:      "errors",
		ShortCommand: "err",
		Usage: `
Usage: errors [count]

This command displays recent error and warning messages from the log.
Default count is 10.
		`,
		Help:             `Show recent error messages from log`,
		Process:          Errors_Process,
		AutoCompleteFunc: nil,
	}
}

func Errors_Process(c *Command, input string) {
	// Parse count argument
	count := 10
	p := cmn.Split3(input)
	if p[1] != "" {
		if n, err := strconv.Atoi(p[1]); err == nil && n > 0 {
			count = n
		}
	}

	// Read log file
	file, err := os.Open(cmn.LogPath)
	if err != nil {
		ui.PrintErrorf("Failed to open log file: %v\n", err)
		return
	}
	defer file.Close()

	// Collect error/warning lines
	var errorLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// zerolog ConsoleWriter format includes ANSI color codes:
		// [31mERR[0m for errors (red), [33mWRN[0m for warnings (yellow)
		if strings.Contains(line, "ERR") || strings.Contains(line, "WRN") {
			errorLines = append(errorLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		ui.PrintErrorf("Failed to read log file: %v\n", err)
		return
	}

	if len(errorLines) == 0 {
		ui.Printf("No errors or warnings in log\n")
		return
	}

	// Get last N lines
	start := 0
	if len(errorLines) > count {
		start = len(errorLines) - count
	}

	width := ui.GetTerminalWidth() - 2 // small margin for safety
	if width < 20 {
		width = 78 // fallback for unreasonable width
	}

	ui.Printf("Recent errors/warnings (%d of %d):\n\n", len(errorLines)-start, len(errorLines))
	for _, line := range errorLines[start:] {
		// Wrap line by terminal width, preserving ANSI codes
		wrapped := wrapLineWithANSI(line, width)
		ui.Printf("%s\n", wrapped)
	}
}

// visibleLength returns the visible length of a string, excluding ANSI escape codes
func visibleLength(s string) int {
	return len(ansiRegex.ReplaceAllString(s, ""))
}

// wrapLineWithANSI wraps a line at the specified width, preserving ANSI escape codes
func wrapLineWithANSI(line string, width int) string {
	if visibleLength(line) <= width {
		return line
	}

	var result strings.Builder
	var currentLineLen int
	var currentANSI string // Track current ANSI state for continuation lines

	i := 0
	for i < len(line) {
		// Check for ANSI escape sequence
		if line[i] == '\x1b' && i+1 < len(line) && line[i+1] == '[' {
			// Find end of escape sequence
			end := i + 2
			for end < len(line) && line[end] != 'm' {
				end++
			}
			if end < len(line) {
				end++ // include 'm'
				ansiCode := line[i:end]
				result.WriteString(ansiCode)
				currentANSI = ansiCode
				i = end
				continue
			}
		}

		// Regular character
		if currentLineLen >= width {
			result.WriteString("\n")
			// Re-apply current ANSI state on new line
			if currentANSI != "" && currentANSI != "\x1b[0m" {
				result.WriteString(currentANSI)
			}
			currentLineLen = 0
		}

		result.WriteByte(line[i])
		currentLineLen++
		i++
	}

	return result.String()
}
