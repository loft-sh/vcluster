package str

import (
	"fmt"
)

// IsEmpty tests for str == ""
func IsEmpty(str string) bool {
	return String(str).IsEmpty()
}

// SplitLines splits each line from str into []string
func SplitLines(str string) []string {
	return String(str).SplitLines()
}

// SplitSpaces splits str by blank chars (space,\t,\n)
func SplitSpaces(str string) []string {
	return String(str).SplitSpaces()
}

// Bool returns the bool equivalent of str ("true" = true, etc)
// A parsing error will cause a program panic.
func Bool(str string) bool {
	s := String(str)
	result := s.Bool()
	if s.Err() != nil {
		panic(fmt.Sprintf("%s", s.Err()))
	}
	return result
}

// Int returns the int representation of str.
// A parsing error will cause a program panic.
func Int(str string) int {
	s := String(str)
	result := s.Int()
	if s.Err() != nil {
		panic(fmt.Sprintf("%s", s.Err()))
	}
	return result
}

// Float64 returns the float64 representation of str.
// A parsing error will cause a program panic.
func Float64(str string) float64 {
	s := String(str)
	result := s.Float64()
	if s.Err() != nil {
		panic(fmt.Sprintf("%s", s.Err()))
	}
	return result
}
