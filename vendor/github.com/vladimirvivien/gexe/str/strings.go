package str

import (
	"bytes"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/vladimirvivien/gexe/vars"
)

var (
	notSpaceRegex = regexp.MustCompile(`\S`)
)

// Str represents a string value
type Str struct {
	val  string
	err  error
	vars *vars.Variables
}

// String is constructor function that returns *Str
func String(str string) *Str {
	return &Str{val: str, vars: &vars.Variables{}}
}

// StringWithVars sets session variables and calls func String
func StringWithVars(str string, variables *vars.Variables) *Str {
	s := String(variables.Eval(str))
	s.vars = variables
	return s
}

// String returns the string value
func (s *Str) String() string {
	return s.val
}

// Err returns any captured error
func (s *Str) Err() error {
	return s.err
}

// IsEmpty returns true if len(s) == 0
func (s *Str) IsEmpty() bool {
	return s.val == ""
}

// Eq returns true if both strings are equal
func (s *Str) Eq(val1 string) bool {
	return strings.EqualFold(s.val, val1)
}

// Split s.val using the sep as delimiter
func (s *Str) Split(sep string) []string {
	return strings.Split(s.val, sep)
}

// SplitLines splits s.val using \n as delimiter
func (s *Str) SplitLines() []string {
	return strings.Split(s.val, "\n")
}

// SplitSpaces properly splits s.val into []elements
// separated by one or more Unicode.IsSpace characters
// i.e. SplitSpaces("ab   cd e\tf\ng") returns 5 elements
func (s *Str) SplitSpaces() []string {
	return notSpaceRegex.Split(s.val, -1)
}

// SplitRegex uses regular expression exp to split s.val
func (s *Str) SplitRegex(exp string) []string {
	return regexp.MustCompile(exp).Split(s.val, -1)
}

// Bytes returns []byte(s.val)
func (s *Str) Bytes() []byte {
	return []byte(s.val)
}

// Bool converts s.val from string to a bool representation
// Check s.Err() for parsing errors
func (s *Str) Bool() bool {
	val, err := strconv.ParseBool(s.val)
	if err != nil {
		s.err = err
	}
	return val
}

// Int converts s.val from string to a int representation
// Check s.Err() for parsing errors
func (s *Str) Int() int {
	val, err := strconv.Atoi(s.val)
	if err != nil {
		s.err = err
	}
	return val
}

// Float64 converts s.val from string to a float64 representation
// Check s.Error() for parsing errors
func (s *Str) Float64() float64 {
	val, err := strconv.ParseFloat(s.val, 64)
	if err != nil {
		s.err = err
	}
	return val
}

// Reader returns an io.Reader to access the content.
func (s *Str) Reader() io.Reader {
	return bytes.NewReader([]byte(s.val))
}

// ToLower returns val as lower case
func (s *Str) ToLower() *Str {
	s.val = strings.ToLower(s.val)
	return s
}

// ToUpper returns val as upper case
func (s *Str) ToUpper() *Str {
	s.val = strings.ToUpper(s.val)
	return s
}

// ToTitle returns strings.ToTitle for s.val
func (s *Str) ToTitle() *Str {
	s.val = strings.ToTitle(s.val)
	return s
}

// TrimSpaces removes spaces around a val
func (s *Str) TrimSpaces() *Str {
	s.val = strings.TrimSpace(s.val)
	return s
}

// TrimLeft removes each character in cutset at the
// start of s.val
func (s *Str) TrimLeft(cutset string) *Str {
	s.val = strings.TrimLeft(s.val, cutset)
	return s
}

// TrimRight removes each character in cutset removed at the
// start of s.val
func (s *Str) TrimRight(cutset string) *Str {
	s.val = strings.TrimRight(s.val, cutset)
	return s
}

// Trim removes each character in cutset from around s.val
func (s *Str) Trim(cutset string) *Str {
	s.val = strings.Trim(s.val, cutset)
	return s
}

// ReplaceAll replaces all occurrences of old with new in s.val
func (s *Str) ReplaceAll(old, new string) *Str {
	s.val = strings.ReplaceAll(s.val, old, new)
	return s
}

// Concat concatenates val1 to s.val
func (s *Str) Concat(vals ...string) *Str {
	evals := []string{s.val}
	for _, val := range vals {
		evals = append(evals, s.vars.Eval(val))
	}
	s.val = strings.Join(evals, "")
	return s
}

// CopyTo copies s.val unto dest
// Check s.Error() for copy error.
func (s *Str) CopyTo(dest io.Writer) *Str {
	if _, err := io.Copy(dest, bytes.NewBufferString(s.val)); err != nil {
		s.err = err
	}
	return s
}
