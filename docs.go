package gaze

import (
	"regexp"
	"strings"
)

var docRules = []struct {
	pattern *regexp.Regexp
	replace string
}{
	{regexp.MustCompile(`(?i)\b(please|kindly|make sure to|remember to)\s+`), ""},
	{regexp.MustCompile(`(?i)\bin order to\b`), "to"},
	{regexp.MustCompile(`(?i)\bdue to the fact that\b`), "because"},
	{regexp.MustCompile(`(?i)\bat this point in time\b`), "now"},
	{regexp.MustCompile(`(?i)\bfor the purpose of\b`), "for"},
	{regexp.MustCompile(`(?i)\bin the event that\b`), "if"},
	{regexp.MustCompile(`(?i)\ba large number of\b`), "many"},
	{regexp.MustCompile(`(?i)\bhas the ability to\b`), "can"},
	{regexp.MustCompile(`(?i)\bis able to\b`), "can"},
	{regexp.MustCompile(`(?i)\bis responsible for\b`), "handles"},
	{regexp.MustCompile(`(?i)\butilize\b`), "use"},
	{regexp.MustCompile(`(?i)\binitiate\b`), "start"},
	{regexp.MustCompile(`(?i)\bterminate\b`), "end"},
	{regexp.MustCompile(`(?i)\bprior to\b`), "before"},
	{regexp.MustCompile(`(?i)\bsubsequent to\b`), "after"},
	{regexp.MustCompile(`(?i)\b(a|an|the|really|basically|actually|essentially)\b`), ""},
	{regexp.MustCompile(`[ \t]{2,}`), " "},
	{regexp.MustCompile(` +([,.;:!?])`), "$1"},
}

var docSingleQuote = regexp.MustCompile(`(^|[^\pL\pN_])'[^'\n]+'([^\pL\pN_]|$)`)

// compactDoc shortens selected English expressions in plain prose only.
func compactDoc(text string) string {
	// ponytail: Skip marked-up or code-like text; extend parsing when richer compression is needed.
	if strings.ContainsAny(text, "`\"<>@{}[]()\\/=:;") || docSingleQuote.MatchString(text) || strings.Contains(text, "    ") {
		return text
	}
	for _, rule := range docRules {
		text = rule.pattern.ReplaceAllString(text, rule.replace)
	}
	return strings.TrimSpace(text)
}
