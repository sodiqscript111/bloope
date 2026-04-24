package zeroreg

import (
	"fmt"
	"regexp"
	"strings"
)

type Pattern struct {
	source string
}

func New(source string) *Pattern {
	return &Pattern{source: source}
}

func StartOfLine() *Pattern {
	return New("^")
}

func EndOfLine() *Pattern {
	return New("$")
}

func Literal(value string) *Pattern {
	return New(regexp.QuoteMeta(value))
}

func CharIn(chars string) *Pattern {
	replacer := strings.NewReplacer(
		`]`, `\]`,
		`\`, `\\`,
		`^`, `\^`,
		`-`, `\-`,
	)

	return New(fmt.Sprintf("[%s]", replacer.Replace(chars)))
}

func Group(pattern *Pattern) *Pattern {
	return New(fmt.Sprintf("(?:%s)", pattern.source))
}

func (p *Pattern) Then(next *Pattern) *Pattern {
	return New(p.source + next.source)
}

func (p *Pattern) ThenStr(value string) *Pattern {
	return p.Then(Literal(value))
}

func (p *Pattern) OneOrMore() *Pattern {
	return New(p.wrap() + "+")
}

func (p *Pattern) Optional() *Pattern {
	return New(p.wrap() + "?")
}

func (p *Pattern) Test(input string) bool {
	return regexp.MustCompile(p.source).MatchString(input)
}

func (p *Pattern) wrap() string {
	if len(p.source) == 1 ||
		strings.HasPrefix(p.source, "\\") ||
		(strings.HasPrefix(p.source, "[") && strings.HasSuffix(p.source, "]")) ||
		(strings.HasPrefix(p.source, "(?:") && strings.HasSuffix(p.source, ")")) {
		return p.source
	}

	return fmt.Sprintf("(?:%s)", p.source)
}
