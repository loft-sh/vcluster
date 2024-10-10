package edlib

import (
	"math"
)

// QgramDistance compute the q-gram similarity between two strings
// Takes two strings as parameters, a split length which defines the k-gram shingle length
func QgramDistance(str1, str2 string, splitLength int) int {
	splittedStr1 := Shingle(str1, splitLength)
	splittedStr2 := Shingle(str2, splitLength)

	union := make(map[string]int)
	for i := range splittedStr1 {
		union[i] = 0
	}
	for i := range splittedStr2 {
		union[i] = 0
	}

	res := 0

	for i := range union {
		res += int(math.Abs(float64(splittedStr1[i] - splittedStr2[i])))
	}

	return res
}

// QgramDistanceCustomNgram compute the q-gram similarity between two custom set of individuals
// Takes two n-gram map as parameters
func QgramDistanceCustomNgram(splittedStr1, splittedStr2 map[string]int) int {
	union := make(map[string]int)
	for i := range splittedStr1 {
		union[i] = 0
	}
	for i := range splittedStr2 {
		union[i] = 0
	}

	res := 0
	for i := range union {
		res += int(math.Abs(float64(splittedStr1[i] - splittedStr2[i])))
	}

	return res
}

// QgramSimilarity compute a similarity index (between 0 and 1) between two strings from a Qgram distance
// Takes two strings as parameters, a split length which defines the k-gram shingle length
func QgramSimilarity(str1, str2 string, splitLength int) float32 {
	splittedStr1 := Shingle(str1, splitLength)
	splittedStr2 := Shingle(str2, splitLength)
	res := float32(QgramDistanceCustomNgram(splittedStr1, splittedStr2))
	return 1 - (res / float32(len(splittedStr1)+len(splittedStr2)))
}
