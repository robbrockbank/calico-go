package multidict

type StringToString map[string]map[string]bool

func NewStringToString() StringToString {
	sToS := make(StringToString)
	return sToS
}

func (s2s StringToString) Put(key, value string) {
	set, ok := s2s[key]
	if !ok {
		set = make(map[string]bool)
		s2s[key] = set
	}
	set[value] = true
}

func (s2s StringToString) Discard(key, value string) {
	set, ok := s2s[key]
	if !ok {
		return
	}
	delete(set, value)
	if len(set) == 0 {
		delete(s2s, key)
	}
}

func (s2s StringToString) Contains(key, value string) bool {
	set, ok := s2s[key]
	return ok && set[value]
}

func (s2s StringToString) ContainsKey(key string) bool {
	_, ok := s2s[key]
	return ok
}

func (s2s StringToString) Iter(key string, f func(value string)) {
	for value, _ := range(s2s[key]) {
		f(value)
	}
}