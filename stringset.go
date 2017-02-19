package treeloader

import "strings"

type StringSet map[string]int

func (ss StringSet) String() string {
	out := make([]string, 0, len(ss))
	for s := range ss {
		out = append(out, s)
	}
	return strings.Join(out, "\n")
}

func (ss StringSet) Set(value string) error {
	for _, s := range strings.Split(value, ",") {
		ss[s] = 1
	}
	return nil
}

func (ss StringSet) Contains(s string) bool {
	_, ok := ss[s]
	return ok
}

func (ss StringSet) Add(sColl ...string) {
	for _, s := range sColl {
		ss[s] = 1
	}
}

func (ss StringSet) Remove(sColl ...string) {
	for _, s := range sColl {
		delete(ss, s)
	}
}
