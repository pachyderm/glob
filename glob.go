package glob

import (
	"fmt"
	"regexp"

	"github.com/glenn-brown/golang-pkg-pcre/src/pkg/pcre"

	"github.com/pachyderm/glob/compiler"
	"github.com/pachyderm/glob/syntax"
	"github.com/pachyderm/glob/syntax/ast"
)

// Glob represents compiled glob pattern.
type Glob struct {
	r *regexp.Regexp
	p *pcre.Regexp
}

// Compile creates Glob for given pattern and strings (if any present after pattern) as separators.
// The pattern syntax is:
//
//    pattern:
//        { term }
//
//    term:
//        `*`         matches any sequence of non-separator characters
//        `**`        matches any sequence of characters
//        `?`         matches any single non-separator character
//        `[` [ `!` ] { character-range } `]`
//                    character class (must be non-empty)
//        `{` pattern-list `}`
//                    pattern alternatives
//        c           matches character c (c != `*`, `**`, `?`, `\`, `[`, `{`, `}`)
//        `\` c       matches character c
//
//    character-range:
//        c           matches character c (c != `\\`, `-`, `]`)
//        `\` c       matches character c
//        lo `-` hi   matches character c for lo <= c <= hi
//
//    pattern-list:
//        pattern { `,` pattern }
//                    comma-separated (without spaces) patterns
//
//    extended-glob:
//        `(` { `|` pattern } `)`
//        `@(` { `|` pattern } `)`
//                    capture one of pipe-separated subpatterns
//        `*(` { `|` pattern } `)`
//                    capture any number of of pipe-separated subpatterns
//        `+(` { `|` pattern } `)`
//                    capture one or more of of pipe-separated subpatterns
//        `?(` { `|` pattern } `)`
//                    capture zero or one of of pipe-separated subpatterns
//
func Compile(pattern string, separators ...rune) (*Glob, error) {
	tree, compilerToUse, err := syntax.Parse(pattern)
	if err != nil {
		return nil, err
	}

	regex, err := compiler.Compile(tree, separators)
	if err != nil {
		return nil, err
	}
	fmt.Println("pattern:", pattern)
	fmt.Println("regexp:", regex, compilerToUse)

	switch compilerToUse {
	case ast.Regexp:
		r, err := regexp.Compile(regex)
		if err != nil {
			return nil, err
		}
		return &Glob{r: r}, nil
	case ast.PCRE:
		p, pcreErr := pcre.Compile(regex, 0)
		if pcreErr != nil {
			return nil, fmt.Errorf(pcreErr.String())
		}
		return &Glob{p: &p}, nil
	default:
		return nil, fmt.Errorf("Unrecognized compiler: %v", compilerToUse)
	}
}

// MustCompile is the same as Compile, except that if Compile returns error, this will panic
func MustCompile(pattern string, separators ...rune) *Glob {
	g, err := Compile(pattern, separators...)
	if err != nil {
		panic(err)
	}
	return g
}

func (g *Glob) Match(fixture string) bool {
	if g.r != nil {
		return g.r.MatchString(fixture)
	}
	m := g.p.MatcherString(fixture, 0)
	return m.MatchString(fixture, 0)
}

func (g *Glob) Capture(fixture string) []string {
	if g.r != nil {
		return g.r.FindStringSubmatch(fixture)
	}
	m := g.p.MatcherString(fixture, 0)
	num := m.Groups()
	groups := make([]string, 0, num)
	if m.MatchString(fixture, 0) {
		for i := 0; i <= num; i++ {
			groups = append(groups, m.GroupString(i))
		}
	}
	return groups
}

// QuoteMeta returns a string that quotes all glob pattern meta characters
// inside the argument text; For example, QuoteMeta(`*(foo*)`) returns `\*\(foo\*\)`.
func QuoteMeta(s string) string {
	b := make([]byte, 2*len(s))

	// a byte loop is correct because all meta characters are ASCII
	j := 0
	for i := 0; i < len(s); i++ {
		if syntax.Special(s[i]) {
			b[j] = '\\'
			j++
		}
		b[j] = s[i]
		j++
	}

	return string(b[0:j])
}
