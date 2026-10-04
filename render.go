package main

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

// Colours are plain ANSI so the terminal theme decides the actual palette.
var (
	accent = lipgloss.Color("4")
	dim    = lipgloss.Color("8")
	danger = lipgloss.Color("1")
	good   = lipgloss.Color("2")

	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	markerStyle  = lipgloss.NewStyle().Foreground(accent)
	quoteStyle   = lipgloss.NewStyle().Foreground(dim).Italic(true)
	codeStyle    = lipgloss.NewStyle().Foreground(dim)
	ruleStyle    = lipgloss.NewStyle().Foreground(dim)
	boldStyle    = lipgloss.NewStyle().Bold(true)
	inlineCode   = lipgloss.NewStyle().Foreground(accent)
)

var (
	reHeading = regexp.MustCompile(`^#{1,6}\s+(.*)$`)
	reRule    = regexp.MustCompile(`^\s*([-*_])(\s*[-*_]){2,}\s*$`)
	reBullet  = regexp.MustCompile(`^(\s*)[-*+]\s+`)
	reNumber  = regexp.MustCompile(`^(\s*)(\d+)[.)]\s+`)
	reCode    = regexp.MustCompile("`([^`]+)`")
	reBold    = regexp.MustCompile(`\*\*(.+?)\*\*`)
)

// renderMarkdown turns a note body into styled, word-wrapped terminal text.
// It handles the subset of markdown that shows up in quick notes: headings,
// bullets, numbered lists, quotes, rules, code fences, bold and inline code.
func renderMarkdown(body string, width int) string {
	if width < 10 {
		width = 10
	}
	var out []string
	inCode := false
	for _, raw := range strings.Split(strings.TrimRight(sanitize(body), "\n"), "\n") {
		line := strings.TrimRight(raw, " \t")
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
			out = append(out, codeStyle.Render(strings.TrimSpace(line)))
			continue
		}
		if inCode {
			out = append(out, codeStyle.Render(line))
			continue
		}
		switch {
		case reRule.MatchString(line):
			out = append(out, ruleStyle.Render(strings.Repeat("─", width)))
		case reHeading.MatchString(line):
			out = append(out, wordwrap.String(headingStyle.Render(reHeading.FindStringSubmatch(line)[1]), width))
		case reBullet.MatchString(line):
			m := reBullet.FindStringSubmatch(line)
			out = append(out, hang(m[1]+markerStyle.Render("•")+" ", inline(line[len(m[0]):]), width))
		case reNumber.MatchString(line):
			m := reNumber.FindStringSubmatch(line)
			out = append(out, hang(m[1]+markerStyle.Render(m[2]+".")+" ", inline(line[len(m[0]):]), width))
		case strings.HasPrefix(line, ">"):
			text := strings.TrimSpace(strings.TrimPrefix(line, ">"))
			out = append(out, hang(quoteStyle.Render("│ "), quoteStyle.Render(text), width))
		default:
			out = append(out, wordwrap.String(inline(line), width))
		}
	}
	return strings.Join(out, "\n")
}

// inline styles `code` and **bold** spans.
func inline(s string) string {
	s = reCode.ReplaceAllStringFunc(s, func(m string) string { return inlineCode.Render(strings.Trim(m, "`")) })
	s = reBold.ReplaceAllStringFunc(s, func(m string) string { return boldStyle.Render(strings.TrimSuffix(strings.TrimPrefix(m, "**"), "**")) })
	return s
}

// hang wraps text after prefix and indents continuation lines under it.
func hang(prefix, text string, width int) string {
	pw := lipgloss.Width(prefix)
	w := width - pw
	if w < 10 {
		w = 10
	}
	lines := strings.Split(wordwrap.String(text, w), "\n")
	pad := strings.Repeat(" ", pw)
	for i, l := range lines {
		if i == 0 {
			lines[i] = prefix + l
		} else {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}
