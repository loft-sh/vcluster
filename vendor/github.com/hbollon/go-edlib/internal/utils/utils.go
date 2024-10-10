package utils

// StringHashMap is HashMap substitute for string
type StringHashMap map[string]struct{}

// Min return the smallest integer among the two in parameters
func Min(a int, b int) int {
	if b < a {
		return b
	}
	return a
}

// Max return the largest integer among the two in parameters
func Max(a int, b int) int {
	if b > a {
		return b
	}
	return a
}

// Equal compare two rune arrays and return if they are equals or not
func Equal(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

/*
	StringHashMap methods
*/

// AddAll adds all elements from one StringHashMap to another
func (m StringHashMap) AddAll(srcMap StringHashMap) {
	for key := range srcMap {
		m[key] = struct{}{}
	}
}

// ToArray convert and return an StringHashMap to string array
func (m StringHashMap) ToArray() []string {
	var index int
	arr := make([]string, 0, len(m))
	for key := range m {
		arr = append(arr, key)
		index++
	}

	return arr
}
