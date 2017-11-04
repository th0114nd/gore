package gore

import (
	"./stringset"
	"github.com/kr/pretty"
	"testing"
)

func TestAcceptor(t *testing.T) {
type data struct {
	a     Acceptor
	input string
	want  stringset.Set
}

	mk := stringset.New

	d := &Dot{}
	s := &Single{'&'}
	r := &Range{'a', 'f'}
	md := &Maybe{d}
	mr := &Maybe{r}
	mmr := &Maybe{mr}
	sd := &Star{&Dot{}}
	sr := &Star{r}
	pr := &Plus{r}
	ss := &Star{&Single{'&'}}
	sq := Sequence{&Range{'g', 't'}, &Range{'h', 'u'}}
	p := &Pipe{left: &Single{'{'}, right: sq}
	c := &Count{&Single{'@'}, 3, 5}
	for _, datum := range []data{
		{d, "", mk()},
		{d, "t", mk("")},
		{d, "t. hanks", mk(". hanks")},
		{s, "", mk()},
		{s, "^", nil},
		{s, "&", mk("")},
		{s, "&cd", mk("cd")},
		{r, "", nil},
		{r, "a", mk("")},
		{r, "f", mk("")},
		{r, "bear", mk("ear")},
		{r, "grab", nil},
		{md, "", mk("")},
		{md, "x", mk("", "x")},
		{md, "zero", mk("ero", "zero")},
		{mr, "", mk("")},
		{mr, "crash", mk("rash", "crash")},
		{mr, "zone", mk("zone")},
		{mmr, "", mk("")},
		{mmr, "hello", mk("hello")},
		{mmr, "abc", mk("bc", "abc")},
		{sd, "", mk("")},
		{sd, "#", mk("", "#")},
		{sd, "hello", mk("", "o", "lo", "llo", "ello", "hello")},
		{sr, "xany", mk("xany")},
		{sr, "always", mk("lways", "always")},
		{sr, "deadgolf", mk("golf", "dgolf", "adgolf", "eadgolf", "deadgolf")},
		{pr, "random", nil},
		{pr, "abc", mk("bc", "c", "")},
		{ss, "*", mk("*")},
		{ss, "&&((", mk("&&((", "&((", "((")},
		{sq, "ghost", mk("ost")},
		{p, "{zztop", mk("zztop")},
		{p, "ghost", mk("ost")},
		{p, "zero", nil},
		{c, "", nil},
		{c, "@@", nil},
		{c, "@@@", mk("")},
		{c, "@@@H@@", mk("H@@")},
		{c, "@@@@@@@@", mk("@@@", "@@@@", "@@@@@")},
	} {
		got := datum.a.Accept(datum.input)
		if diff := pretty.Diff(got, datum.want); diff != nil {
			t.Errorf("%q.Accept(%q) = %q, want %q\n%v", datum.a, datum.input, got, datum.want, diff)
		}
	}

}

func TestParseRange(t *testing.T) {
	type data struct {
		input string
		end   int
		want  Acceptor
	}
	for _, datum := range []data{
		{"a]garbage", 2, &Single{'a'}},
		{"p-y]", 4, &Range{'p', 'y'}},
		{"]g]x]", 3, &Pipe{&Single{']'}, &Single{'g'}}},
		// {"4-57-8-]", 8, &Pipe{&Pipe{&Range{4, 5}, &Range{7, 8}}, &Pipe{']'}}},
	} {
		got, gotEnd := parseRange(datum.input)
		if gotEnd != datum.end {
			t.Errorf("parseRange(%v).end = %v, want %v", datum.input, gotEnd, datum.end)
		}
		if diff := pretty.Diff(got, datum.want); diff != nil {
			t.Errorf("parseRange(%v) = %v, want %v", datum.input, got, datum.want)
		}

	}
}

func TestParse(t *testing.T) {
	type data struct {
		input string
		want  Regexp
	}
	for _, datum := range []data{
		{"", nil},
		{".", []Acceptor{&Dot{}}},
		{".*", []Acceptor{&Star{&Dot{}}}},
		{".**", []Acceptor{&Star{&Star{&Dot{}}}}},
		{"awk+", []Acceptor{&Single{'a'}, &Single{'w'}, &Plus{&Single{'k'}}}},
		{"x*", []Acceptor{&Star{&Single{'x'}}}},
		{"z?", []Acceptor{&Maybe{&Single{'z'}}}},
		{"[0-9]", []Acceptor{&Range{'0', '9'}}},
		{"a|b", []Acceptor{&Pipe{Sequence{&Single{'a'}}, Sequence{&Single{'b'}}}}},
		{"45|4*", []Acceptor{&Pipe{Sequence{&Single{'4'}, &Single{'5'}},
			Sequence{&Star{&Single{'4'}}}}}},
		{"[a.*]", []Acceptor{&Pipe{&Pipe{&Single{'a'}, &Single{'.'}}, &Single{'*'}}}},
		{`\*`, []Acceptor{&Single{'*'}}},
		{"b{8,100}", []Acceptor{&Count{&Single{'b'}, 8, 100}}},
	} {
		got, err := Parse(datum.input)
		if err != nil {
			t.Errorf("Parse(%v) errored: %v", datum.input, err)
		}
		if diff := pretty.Diff(got, datum.want); diff != nil {
			t.Errorf("Parse(%v) = %v, want %v", datum.input, got, datum.want)
		}
	}

}

func TestMatch(t *testing.T) {
	type data struct {
		re    string
		input string
		want  bool
	}
	for _, datum := range []data{
		{"", "", true},
		{"", "nonempty", true},
		{".*", "anything, really", true},
		{"x*", "xxxxx", true},
		{"x*", "y", true},
		{"x+", "y", false},
		{"&?c", "&c", true},
		{"&?c", "c", true},
		{"&?c", "b", false},
		{".{4,4}", "acbd", true},
		{".{4,4}", "ggg", false},
		{".{4,4}", "*****", true},
	} {
		r, err := Parse(datum.re)
		if err != nil {
			t.Errorf("Parse(%v) errored: %v", datum.re, err)
		}
		if got := r.Match(datum.input); got != datum.want {
			t.Errorf("%v.Match(%v) = %v, want %v", r, datum.input, got, datum.want)
		}
	}
}
