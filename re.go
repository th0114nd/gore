package gore

// metachars {}[]()^$.|*+?\
// . -> one character (lets say ascii to start)
//     may not match \n
//     [a.c] matches x \in 'a', '.', 'c'}
// TODO ^ -> beginning of a string
// TODO $ -> end of a string
//     line oriented: just before \n
// | -> alternatives between neighbors
//     What's its precedence? v low
// * -> arbitrary repitition of predecessor
// + -> positive repetition of predecessor
// ? -> maybe match of predecessor
// \ -> escape character
// [...] -> character class
// TODO     ^ implies negation
//     x-y implies a range
//     - a literal if last or first
//     ] can be only matched if it is first (perhaps after ^)
// {m, n} -> counts of repitition
//     matches previous at least m and at most n times
// TODO     {m,m} <=> {m}
// TODO () -> marks a subexpression
//     changes precedence, and marks this pattern with a number 1-9
// TODO \1, \2, .. \9 -> matches the nth marked subexpression
// TODO return the longest match instead of whether there was a match

import (
	"./stringset"
	"errors"
	"fmt"
)

type Acceptor interface {
	fmt.Stringer
	Accept(cs string) stringset.Set
}

type Dot struct{}

func (Dot) String() string { return "/./" }
func (Dot) Accept(cs string) stringset.Set {
	if len(cs) == 0 {
		return nil
	}
	if cs[0] == 0 || cs[0] >= 0x80 {
		return nil
	}
	return stringset.New(cs[1:])
}

type Single struct{ c byte }

func (s *Single) String() string { return fmt.Sprintf("/%v/", s.c) }
func (s *Single) Accept(cs string) stringset.Set {
	if len(cs) == 0 {
		return nil
	}
	if s.c != cs[0] {
		return nil
	}
	return stringset.New(cs[1:])
}

type Range struct {
	lo byte
	hi byte
}

func (r *Range) String() string { return fmt.Sprintf("/[%v-%v]/", r.lo, r.hi) }
func (r *Range) Accept(cs string) stringset.Set {
	if len(cs) == 0 {
		return nil
	}
	if cs[0] < r.lo || r.hi < cs[0] {
		return nil
	}
	return stringset.New(cs[1:])
}

type Maybe struct{ mini Acceptor }

func (m *Maybe) String() string { return fmt.Sprintf("/%v?/", m.mini) }
func (m *Maybe) Accept(cs string) stringset.Set {
	set := stringset.New(cs)
	bs := m.mini.Accept(cs)
	for b := range bs {
		set.Add(b)
	}
	return set
}

type Star struct{ mini Acceptor }

func (s *Star) String() string { return fmt.Sprintf("/%v*/", s.mini) }

// p (abc) -> "ab
// We have a set of strings
// call Accept on any one of them gives a new set
// if any are already found, do not Accept them.
// Otherwise, call Accept on the next one and continue
func (st *Star) Accept(cs string) stringset.Set {
	set := stringset.New(cs)
	var popd string
	queue := []string{cs}
	for len(queue) > 0 {
		popd, queue = queue[0], queue[1:]
		next := st.mini.Accept(popd)
		for n := range next {
			if set.Has(n) {
				continue
			}
			set.Add(n)
			queue = append(queue, n)
		}
	}
	return set
}

type Count struct {
	mini Acceptor
	min  int
	max  int
}

func (c *Count) String() string { return fmt.Sprintf("/%v{%d,%d}/", c.mini, c.min, c.max) }

func (c *Count) Accept(cs string) stringset.Set {
	depths := map[int]stringset.Set{0: stringset.New(cs)}
	// This algorithm seems horrendously inefficient.
	// The results of c.mini.Accept should be cached.
	for i := 0; i < c.max; i++ {
		tmp := stringset.New()
		for s := range depths[i] {
			tmp.Union(c.mini.Accept(s))
		}
		depths[i+1] = tmp
	}
	out := stringset.New()
	for i := c.min; i <= c.max; i++ {
		out.Union(depths[i])
	}
	return out
}

type Plus struct{ mini Acceptor }

func (p *Plus) String() string { return fmt.Sprintf("/%v+/", p.mini) }
func (p *Plus) Accept(cs string) stringset.Set {
	// Accept 0 or more repititions of the next possible states.
	next := p.mini.Accept(cs)
	out := stringset.New()
	star := &Star{p.mini}
	for n := range next {
		children := star.Accept(n)
		for c := range children {
			out.Add(c)
		}
	}
	return out
}

type Sequence []Acceptor

func (s Sequence) String() string { return fmt.Sprintf("/(%v)/", []Acceptor(s)) }
func (s Sequence) Accept(cs string) stringset.Set {
	if len(s) == 0 {
		return stringset.New(cs)
	}
	a, as := s[0], s[1:]
	next := a.Accept(cs)
	out := stringset.New()
	for n := range next {
		out.Union(as.Accept(n))
	}
	return out
}

type Pipe struct {
	left  Acceptor
	right Acceptor
}

func (p *Pipe) String() string { return fmt.Sprintf("/%v|%v/", p.left, p.right) }
func (p *Pipe) Accept(cs string) stringset.Set {
	out := stringset.New()
	out.Union(p.left.Accept(cs))
	out.Union(p.right.Accept(cs))
	return out
}

type Regexp []Acceptor
type PipePlaceHolder struct{}

func (*PipePlaceHolder) String() string { return "you shouldn't be here" }
func (*PipePlaceHolder) Accept(string) stringset.Set {
	panic("re compilation failed")
}

// parseRange expects that '[' has alread been consumed, so we are
// looking for the character classes and the closing brace.
func parseRange(input string) (Acceptor, int) {
	var choices []Acceptor
	end := 0
	if input[0] == ']' {
		// This invalidates [] as an empty character class,
		// and instead we treat the ']' as escaped.
		end = 1
	}
	for ; input[end] != ']'; end++ {
	}
	if end >= len(input) {
		return &Dot{}, -1
	}
	// Input is now between braces.
	input = input[:end]
	if input[0] == '-' || input[len(input)-1] == '-' {
		choices = append(choices, &Single{'-'})
	}
	for i := 0; i < len(input); {
		if i < len(input)-1 && input[i+1] == '-' {
			choices = append(choices, &Range{input[i], input[i+2]})
			i += 3
		} else {
			choices = append(choices, &Single{input[i]})
			i++
		}
	}
	lhs := choices[0]
	// [abcx-z] is the same as fold | [a, b, c x-z], although could be rebalanced.
	for _, rhs := range choices[1:] {
		lhs = &Pipe{lhs, rhs}
	}

	return lhs, end + 1
}

func Parse(input string) (Regexp, error) {
	var out []Acceptor
	var ch byte
Loop:
	for len(input) > 0 {
		end := len(out) - 1
		ch, input = input[0], input[1:]
		switch ch {
		case '.':
			out = append(out, &Dot{})
		case '*':
			if len(out) == 0 {
				return nil, errors.New("syntax error")
			}
			out[end] = &Star{out[end]}
		case '+':
			if len(out) == 0 {
				return nil, errors.New("syntax error")
			}
			out[end] = &Plus{out[end]}
		case '?':
			if len(out) == 0 {
				return nil, errors.New("syntax error")
			}
			out[end] = &Maybe{out[end]}
		case '|':
			right, _ := Parse(input)
			out = []Acceptor{&Pipe{Sequence(out), Sequence(right)}}
			break Loop
		case '{':
			if len(out) == 0 {
				return nil, errors.New("syntax error")
			}
			var min, max int
			_, err := fmt.Sscanf(input, "%d,%d", &min, &max)
			if err != nil {
				return nil, err
			}
			out[end] = &Count{out[end], min, max}
			for input[0] != '}' {
				input = input[1:]
			}
			input = input[1:]
		case '[':
			matcher, bracedx := parseRange(input)
			if bracedx < 0 {
				return nil, errors.New("syntax error")
			}
			out = append(out, matcher)
			input = input[bracedx:]
		case '\\':
			if len(input) == 0 {
				return nil, errors.New("syntax error")
			}
			out = append(out, &Single{input[0]})
			input = input[1:]
		default:
			out = append(out, &Single{ch})
		}
	}
	return out, nil
}

func (r Regexp) Match(input string) bool {
	return len((Sequence)(r).Accept(input)) != 0
}
