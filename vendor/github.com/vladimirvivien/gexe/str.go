package gexe

import "github.com/vladimirvivien/gexe/str"

// String creates a new str.Str value with string manipulation methods
func (e *Echo) String(s string) *str.Str {
	return str.StringWithVars(s, e.vars)
}
