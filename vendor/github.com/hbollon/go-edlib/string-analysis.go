package edlib

import (
	"errors"

	"github.com/hbollon/go-edlib/internal/orderedmap"
)

// Algorithm is an Integer type used to identify edit distance algorithms
type Algorithm uint8

// Algorithm identifiers
const (
	Levenshtein Algorithm = iota
	DamerauLevenshtein
	OSADamerauLevenshtein
	Lcs
	Hamming
	Jaro
	JaroWinkler
	Cosine
	Jaccard
	SorensenDice
	Qgram
)

// StringsSimilarity return a similarity index [0..1] between two strings based on given edit distance algorithm in parameter.
// Use defined Algorithm type.
// Through this function, Cosine and Jaccard algorithms are used with Shingle split method with a length of 2.
func StringsSimilarity(str1 string, str2 string, algo Algorithm) (float32, error) {
	switch algo {
	case Levenshtein:
		return matchingIndex(str1, str2, LevenshteinDistance(str1, str2)), nil
	case DamerauLevenshtein:
		return matchingIndex(str1, str2, DamerauLevenshteinDistance(str1, str2)), nil
	case OSADamerauLevenshtein:
		return matchingIndex(str1, str2, OSADamerauLevenshteinDistance(str1, str2)), nil
	case Lcs:
		return matchingIndex(str1, str2, LCSEditDistance(str1, str2)), nil
	case Hamming:
		distance, err := HammingDistance(str1, str2)
		if err == nil {
			return matchingIndex(str1, str2, distance), nil
		}
		return 0.0, err
	case Jaro:
		return JaroSimilarity(str1, str2), nil
	case JaroWinkler:
		return JaroWinklerSimilarity(str1, str2), nil
	case Cosine:
		return CosineSimilarity(str1, str2, 2), nil
	case Jaccard:
		return JaccardSimilarity(str1, str2, 2), nil
	case SorensenDice:
		return SorensenDiceCoefficient(str1, str2, 2), nil
	case Qgram:
		return QgramSimilarity(str1, str2, 2), nil
	default:
		return 0.0, errors.New("Illegal argument for algorithm method")
	}
}

// Return matching index E [0..1] from two strings and an edit distance
func matchingIndex(str1 string, str2 string, distance int) float32 {
	// Convert strings to rune slices
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)
	// Compare rune arrays length and make a matching percentage between them
	if len(runeStr1) >= len(runeStr2) {
		return float32(len(runeStr1)-distance) / float32(len(runeStr1))
	}
	return float32(len(runeStr2)-distance) / float32(len(runeStr2))
}

// FuzzySearch realize an approximate search on a string list and return the closest one compared
// to the string input
func FuzzySearch(str string, strList []string, algo Algorithm) (string, error) {
	var higherMatchPercent float32
	var tmpStr string
	for _, strToCmp := range strList {
		sim, err := StringsSimilarity(str, strToCmp, algo)
		if err != nil {
			return "", err
		}

		if sim == 1.0 {
			return strToCmp, nil
		} else if sim > higherMatchPercent {
			higherMatchPercent = sim
			tmpStr = strToCmp
		}
	}

	return tmpStr, nil
}

// FuzzySearchThreshold realize an approximate search on a string list and return the closest one compared
// to the string input. Takes a similarity threshold in parameter.
func FuzzySearchThreshold(str string, strList []string, minSim float32, algo Algorithm) (string, error) {
	var higherMatchPercent float32
	var tmpStr string
	for _, strToCmp := range strList {
		sim, err := StringsSimilarity(str, strToCmp, algo)
		if err != nil {
			return "", err
		}

		if sim == 1.0 {
			return strToCmp, nil
		} else if sim > higherMatchPercent && sim >= minSim {
			higherMatchPercent = sim
			tmpStr = strToCmp
		}
	}
	return tmpStr, nil
}

// FuzzySearchSet realize an approximate search on a string list and return a set composed with x strings compared
// to the string input sorted by similarity with the base string.
// Takes the a quantity parameter to define the number of output strings desired
// (For example 3 in the case of the Google Keyboard word suggestion).
func FuzzySearchSet(str string, strList []string, quantity int, algo Algorithm) ([]string, error) {
	sortedMap := make(orderedmap.OrderedMap, quantity)
	for _, strToCmp := range strList {
		sim, err := StringsSimilarity(str, strToCmp, algo)
		if err != nil {
			return nil, err
		}

		if sim > sortedMap[sortedMap.Len()-1].Value {
			sortedMap[sortedMap.Len()-1].Key = strToCmp
			sortedMap[sortedMap.Len()-1].Value = sim
			sortedMap.SortByValues()
		}
	}

	return sortedMap.ToArray(), nil
}

// FuzzySearchSetThreshold realize an approximate search on a string list and return a set composed with x strings compared
// to the string input sorted by similarity with the base string. Take a similarity threshold in parameter.
// Takes the a quantity parameter to define the number of output strings desired
// (For example 3 in the case of the Google Keyboard word suggestion).
// Takes also a threshold parameter for similarity with base string.
func FuzzySearchSetThreshold(str string, strList []string, quantity int, minSim float32, algo Algorithm) ([]string, error) {
	sortedMap := make(orderedmap.OrderedMap, quantity)
	for _, strToCmp := range strList {
		sim, err := StringsSimilarity(str, strToCmp, algo)
		if err != nil {
			return nil, err
		}

		if sim >= minSim && sim > sortedMap[sortedMap.Len()-1].Value {
			sortedMap[sortedMap.Len()-1].Key = strToCmp
			sortedMap[sortedMap.Len()-1].Value = sim
			sortedMap.SortByValues()
		}
	}

	return sortedMap.ToArray(), nil
}
