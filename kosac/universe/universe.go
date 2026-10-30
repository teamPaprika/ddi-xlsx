// Package universe parses and evaluates response-universe and cleansing
// conditions such as "SQ3!=1 AND SQ3_1!=-8" or "SCH=pattern:^[0-9]{7}$".
package universe

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Row gives the raw value of a variable in one data row.
type Row interface {
	Value(name string) (string, bool)
}

// Expr is a parsed condition: an OR of AND groups.
type Expr struct {
	or   [][]cond
	vars []string
}

type cond struct {
	name  string
	neg   bool
	items []item
	re    *regexp.Regexp
}

type item struct {
	s      string
	lo, hi float64
	rng    bool
}

const patternPrefix = "pattern:"

// Parse parses a RESP_UNIVERSE condition.
func Parse(s string) (*Expr, error) {
	return parse(s, "")
}

// ParseCondition parses a cleansing condition. Besides the Parse grammar it
// accepts a bare "pattern:{regex}" as a condition on target.
func ParseCondition(s, target string) (*Expr, error) {
	return parse(s, target)
}

// Vars returns the referenced variable names in first-appearance order.
func (e *Expr) Vars() []string {
	return slices.Clone(e.vars)
}

// Eval evaluates the condition against r. Every condition is evaluated, so a
// variable missing from r is an error even where AND/OR could short-circuit.
func (e *Expr) Eval(r Row) (bool, error) {
	result := false
	for _, group := range e.or {
		all := true
		for _, c := range group {
			ok, err := c.eval(r)
			if err != nil {
				return false, err
			}
			all = all && ok
		}
		result = result || all
	}
	return result, nil
}

func (c cond) eval(r Row) (bool, error) {
	v, ok := r.Value(c.name)
	if !ok {
		return false, fmt.Errorf("변수 %s가 행에 없습니다", c.name)
	}
	return c.match(v) != c.neg, nil
}

func (c cond) match(v string) bool {
	// An empty value equals nothing, so "!=" holds for it.
	if v == "" {
		return false
	}
	if c.re != nil {
		return c.re.MatchString(v)
	}
	n, isNum := number(v)
	for _, it := range c.items {
		if it.rng {
			if isNum && it.lo <= n && n <= it.hi {
				return true
			}
			continue
		}
		if m, ok := number(it.s); ok && isNum {
			if m == n {
				return true
			}
		} else if it.s == v {
			return true
		}
	}
	return false
}

// number keeps NaN and Inf out so "NaN" or "Inf" codes compare as strings.
func number(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
}

type parser struct {
	src    []rune
	pos    int
	target string // non-empty when a bare pattern is allowed
	vars   []string
}

func parse(s, target string) (*Expr, error) {
	p := &parser{src: []rune(s), target: target}
	groups := [][]cond{nil}
	afterPattern := false
	kwAt := 0
	for {
		c, err := p.cond()
		if err != nil {
			if afterPattern {
				// The regex ended at the keyword, but what follows is not a
				// condition, so the keyword most likely belonged to the regex.
				return nil, p.errAt(kwAt, "pattern 안의 AND/OR는 지원하지 않습니다. 정규식에서는 [ ]AND[ ]처럼 쓰세요")
			}
			return nil, err
		}
		if !slices.Contains(p.vars, c.name) {
			p.vars = append(p.vars, c.name)
		}
		last := len(groups) - 1
		groups[last] = append(groups[last], c)

		kw, at, err := p.keyword()
		if err != nil {
			return nil, err
		}
		switch kw {
		case "":
			return &Expr{or: groups, vars: p.vars}, nil
		case "OR":
			groups = append(groups, nil)
		}
		afterPattern, kwAt = c.re != nil, at
	}
}

func (p *parser) cond() (cond, error) {
	p.skipSpace()
	at := p.pos
	switch {
	case p.eof():
		return cond{}, p.errAt(at, "조건이 없습니다")
	case p.src[p.pos] == '(':
		return cond{}, p.errAt(at, "괄호는 지원하지 않습니다")
	case p.target != "" && p.hasPrefix(patternPrefix):
		return p.pattern(p.target)
	}

	name := p.name()
	if name == "" {
		return cond{}, p.errAt(at, "변수 이름은 영문자로 시작해야 합니다")
	}
	p.skipSpace()
	neg := false
	switch {
	case p.hasPrefix("!="):
		neg = true
		p.pos += 2
	case p.hasPrefix("="):
		p.pos++
	default:
		return cond{}, p.errAt(p.pos, "%s 뒤에 = 또는 !=가 와야 합니다", name)
	}
	p.skipSpace()
	if p.hasPrefix(patternPrefix) {
		if neg {
			return cond{}, p.errAt(p.pos, "pattern은 =와만 쓸 수 있습니다")
		}
		return p.pattern(name)
	}

	c := cond{name: name, neg: neg}
	for {
		it, err := p.item()
		if err != nil {
			return cond{}, err
		}
		c.items = append(c.items, it)
		if !p.consumeAfterSpace(',') {
			return c, nil
		}
		p.skipSpace()
	}
}

func (p *parser) item() (item, error) {
	at := p.pos
	lo, err := p.value()
	if err != nil {
		return item{}, err
	}
	if !p.consumeAfterSpace('~') {
		return item{s: lo}, nil
	}
	p.skipSpace()
	hi, err := p.value()
	if err != nil {
		return item{}, err
	}
	l, ok1 := number(lo)
	h, ok2 := number(hi)
	if !ok1 || !ok2 {
		return item{}, p.errAt(at, "범위(~)는 수끼리만 쓸 수 있습니다")
	}
	return item{lo: l, hi: h, rng: true}, nil
}

func (p *parser) value() (string, error) {
	start := p.pos
	for !p.eof() {
		r := p.src[p.pos]
		if unicode.IsSpace(r) || r == ',' || r == '~' {
			break
		}
		if r == '(' || r == ')' {
			return "", p.errAt(p.pos, "괄호는 지원하지 않습니다")
		}
		p.pos++
	}
	if p.pos == start {
		return "", p.errAt(start, "값이 없습니다")
	}
	return string(p.src[start:p.pos]), nil
}

// patternEnd finds AND/OR with whitespace on both sides. The class is exactly
// the set unicode.IsSpace accepts, so a regex ends where keyword() would read
// a keyword.
var patternEnd = regexp.MustCompile(`[\s\v\x{85}\pZ](?:AND|OR)[\s\v\x{85}\pZ]`)

// pattern takes the regex up to the end of input or the first AND/OR between
// whitespace, so a regex may hold spaces and commas but not those two keywords.
// Edge whitespace is dropped: a stray space from a spreadsheet cell would make
// an anchored regex never match, and a meaningful space can be written as [ ].
func (p *parser) pattern(name string) (cond, error) {
	p.pos += len(patternPrefix)
	at := p.pos
	rest := string(p.src[p.pos:])
	end := len(rest)
	if loc := patternEnd.FindStringIndex(rest); loc != nil {
		end = loc[0]
	}
	p.pos += utf8.RuneCountInString(rest[:end])
	src := strings.TrimFunc(rest[:end], unicode.IsSpace)
	if src == "" {
		return cond{}, p.errAt(at, "정규식이 없습니다")
	}
	re, err := regexp.Compile(src)
	if err != nil {
		return cond{}, p.errAt(at, "정규식이 올바르지 않습니다: %v", err)
	}
	return cond{name: name, re: re}, nil
}

// keyword reads the AND/OR after a condition, or "" at the end of input.
// It returns the keyword's position for errors about what follows it.
func (p *parser) keyword() (string, int, error) {
	spaced := p.skipSpace()
	if p.eof() {
		return "", 0, nil
	}
	at := p.pos
	for !p.eof() && !unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
	w := string(p.src[at:p.pos])
	switch {
	case spaced && (w == "AND" || w == "OR"):
	case spaced && (strings.EqualFold(w, "AND") || strings.EqualFold(w, "OR")):
		return "", 0, p.errAt(at, "AND와 OR는 대문자로 써야 합니다")
	default:
		return "", 0, p.errAt(at, "조건 뒤에는 AND나 OR만 올 수 있습니다 (%q)", w)
	}
	if !p.skipSpace() {
		return "", 0, p.errAt(at, "%s 뒤에 조건이 없습니다", w)
	}
	return w, at, nil
}

func (p *parser) name() string {
	start := p.pos
	for !p.eof() {
		r := p.src[p.pos]
		letter := 'A' <= r && r <= 'Z' || 'a' <= r && r <= 'z'
		if !letter && (p.pos == start || !('0' <= r && r <= '9' || r == '_')) {
			break
		}
		p.pos++
	}
	return string(p.src[start:p.pos])
}

// consumeAfterSpace consumes optional spaces and then r; if r is not there,
// it leaves the position unchanged so the spaces can still precede a keyword.
func (p *parser) consumeAfterSpace(r rune) bool {
	save := p.pos
	p.skipSpace()
	if !p.eof() && p.src[p.pos] == r {
		p.pos++
		return true
	}
	p.pos = save
	return false
}

func (p *parser) skipSpace() bool {
	start := p.pos
	for !p.eof() && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
	return p.pos > start
}

func (p *parser) hasPrefix(s string) bool {
	return strings.HasPrefix(string(p.src[p.pos:]), s)
}

func (p *parser) eof() bool { return p.pos >= len(p.src) }

func (p *parser) errAt(pos int, format string, args ...any) error {
	return fmt.Errorf("%d번째 글자: %s", pos+1, fmt.Sprintf(format, args...))
}

// Values returns the single values compared with name by = or !=, in order.
// Ranges and patterns are left out.
func (e *Expr) Values(name string) []string {
	var out []string
	for _, group := range e.or {
		for _, c := range group {
			if c.name != name {
				continue
			}
			for _, it := range c.items {
				if !it.rng {
					out = append(out, it.s)
				}
			}
		}
	}
	return out
}
