package edlib

import (
	"errors"

	"github.com/hbollon/go-edlib/internal/utils"
)

// LCS takes two strings and compute their LCS(Longuest Common Subsequence)
func LCS(str1, str2 string) int {
	// Convert strings to rune array to handle no-ASCII characters
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	if len(runeStr1) == 0 || len(runeStr2) == 0 {
		return 0
	} else if utils.Equal(runeStr1, runeStr2) {
		return len(runeStr1)
	}

	lcsMatrix := lcsProcess(runeStr1, runeStr2)
	return lcsMatrix[len(runeStr1)][len(runeStr2)]
}

// Return computed lcs matrix
func lcsProcess(runeStr1, runeStr2 []rune) [][]int {
	// 2D Array that will contain str1 and str2 LCS
	lcsMatrix := make([][]int, len(runeStr1)+1)
	for i := 0; i <= len(runeStr1); i++ {
		lcsMatrix[i] = make([]int, len(runeStr2)+1)
		for j := 0; j <= len(runeStr2); j++ {
			lcsMatrix[i][j] = 0
		}
	}

	for i := 1; i <= len(runeStr1); i++ {
		for j := 1; j <= len(runeStr2); j++ {
			if runeStr1[i-1] == runeStr2[j-1] {
				lcsMatrix[i][j] = lcsMatrix[i-1][j-1] + 1
			} else {
				lcsMatrix[i][j] = utils.Max(lcsMatrix[i][j-1], lcsMatrix[i-1][j])
			}
		}
	}

	return lcsMatrix
}

// LCSBacktrack returns all choices taken during LCS process
func LCSBacktrack(str1, str2 string) (string, error) {
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	if len(runeStr1) == 0 || len(runeStr2) == 0 {
		return "", errors.New("Can't process and backtrack any LCS with empty string")
	} else if utils.Equal(runeStr1, runeStr2) {
		return str1, nil
	}

	return processLCSBacktrack(str1, str2, lcsProcess(runeStr1, runeStr2), len(runeStr1), len(runeStr2)), nil
}

func processLCSBacktrack(str1, str2 string, lcsMatrix [][]int, m, n int) string {
	// Convert strings to rune array to handle no-ASCII characters
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	if m == 0 || n == 0 {
		return ""
	} else if runeStr1[m-1] == runeStr2[n-1] {
		return processLCSBacktrack(str1, str2, lcsMatrix, m-1, n-1) + string(runeStr1[m-1])
	} else if lcsMatrix[m][n-1] > lcsMatrix[m-1][n] {
		return processLCSBacktrack(str1, str2, lcsMatrix, m, n-1)
	}

	return processLCSBacktrack(str1, str2, lcsMatrix, m-1, n)
}

// LCSBacktrackAll returns an array containing all common substrings between str1 and str2
func LCSBacktrackAll(str1, str2 string) ([]string, error) {
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	if len(runeStr1) == 0 || len(runeStr2) == 0 {
		return nil, errors.New("Can't process and backtrack any LCS with empty string")
	} else if utils.Equal(runeStr1, runeStr2) {
		return []string{str1}, nil
	}

	return processLCSBacktrackAll(str1, str2, lcsProcess(runeStr1, runeStr2), len(runeStr1), len(runeStr2)).ToArray(), nil
}

func processLCSBacktrackAll(str1, str2 string, lcsMatrix [][]int, m, n int) utils.StringHashMap {
	// Convert strings to rune array to handle no-ASCII characters
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	// Map containing all commons substrings (Hash set builded from map)
	substrings := make(utils.StringHashMap)

	if m == 0 || n == 0 {
		substrings[""] = struct{}{}
	} else if runeStr1[m-1] == runeStr2[n-1] {
		for key := range processLCSBacktrackAll(str1, str2, lcsMatrix, m-1, n-1) {
			substrings[key+string(runeStr1[m-1])] = struct{}{}
		}
	} else {
		if lcsMatrix[m-1][n] >= lcsMatrix[m][n-1] {
			substrings.AddAll(processLCSBacktrackAll(str1, str2, lcsMatrix, m-1, n))
		}
		if lcsMatrix[m][n-1] >= lcsMatrix[m-1][n] {
			substrings.AddAll(processLCSBacktrackAll(str1, str2, lcsMatrix, m, n-1))
		}
	}

	return substrings
}

// LCSDiff will backtrack through the lcs matrix and return the diff between the two sequences
func LCSDiff(str1, str2 string) ([]string, error) {
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	if len(runeStr1) == 0 || len(runeStr2) == 0 {
		return nil, errors.New("Can't process LCS diff with empty string")
	} else if utils.Equal(runeStr1, runeStr2) {
		return []string{str1}, nil
	}

	diff := processLCSDiff(str1, str2, lcsProcess(runeStr1, runeStr2), len(runeStr1), len(runeStr2))
	return diff, nil
}

func processLCSDiff(str1 string, str2 string, lcsMatrix [][]int, m, n int) []string {
	// Convert strings to rune array to handle no-ASCII characters
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	diff := make([]string, 2)

	if m > 0 && n > 0 && runeStr1[m-1] == runeStr2[n-1] {
		diff = processLCSDiff(str1, str2, lcsMatrix, m-1, n-1)
		diff[0] = diff[0] + " " + string(runeStr1[m-1])
		diff[1] = diff[1] + "  "
		return diff
	} else if n > 0 && (m == 0 || lcsMatrix[m][n-1] > lcsMatrix[m-1][n]) {
		diff = processLCSDiff(str1, str2, lcsMatrix, m, n-1)
		diff[0] = diff[0] + " " + string(runeStr2[n-1])
		diff[1] = diff[1] + " +"
		return diff
	} else if m > 0 && (n == 0 || lcsMatrix[m][n-1] <= lcsMatrix[m-1][n]) {
		diff = processLCSDiff(str1, str2, lcsMatrix, m-1, n)
		diff[0] = diff[0] + " " + string(runeStr1[m-1])
		diff[1] = diff[1] + " -"
		return diff
	}

	return diff
}

// LCSEditDistance determines the edit distance between two strings using LCS function
// (allow only insert and delete operations)
func LCSEditDistance(str1, str2 string) int {
	if len(str1) == 0 {
		return len(str2)
	} else if len(str2) == 0 {
		return len(str1)
	} else if str1 == str2 {
		return 0
	}

	lcs := LCS(str1, str2)
	return (len([]rune(str1)) - lcs) + (len([]rune(str2)) - lcs)
}
