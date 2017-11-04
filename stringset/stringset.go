package stringset

type Set map[string]struct{}

func New(ss ...string) Set {
    set := make(Set)
    for _, s := range ss {
        set[s] = struct{}{}
    }
    return set
}

func (s Set) Slice() []string {
    var sl []string
    for k := range s {
        sl = append(sl, k)
    }
    return sl
}

func (s Set) Add(k string) {
    s[k] = struct{}{}
}

func (s Set) Has(k string) bool {
    _, ok := s[k]
    return ok
}

func (s Set) Union(t Set) {
    if s == nil {
        s = New()
    }
    for k := range t {
        s[k] = struct{}{}
    }
}
