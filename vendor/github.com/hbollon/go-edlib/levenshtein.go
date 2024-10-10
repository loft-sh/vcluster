package edlib

import "github.com/hbollon/go-edlib/internal/utils"

// LevenshteinDistance calculate the distance between two string
// This algorithm allow insertions, deletions and substitutions to change one string to the second
// Compatible with non-ASCII characters
func LevenshteinDistance(str1, str2 string) int {
	// Convert string parameters to rune arrays to be compatible with non-ASCII
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	// Get and store length of these strings
	runeStr1len := len(runeStr1)
	runeStr2len := len(runeStr2)
	if runeStr1len == 0 {
		return runeStr2len
	} else if runeStr2len == 0 {
		return runeStr1len
	} else if utils.Equal(runeStr1, runeStr2) {
		return 0
	}

	column := make([]int, runeStr1len+1)

	for y := 1; y <= runeStr1len; y++ {
		column[y] = y
	}
	for x := 1; x <= runeStr2len; x++ {
		column[0] = x
		lastkey := x - 1
		for y := 1; y <= runeStr1len; y++ {
			oldkey := column[y]
			var i int
			if runeStr1[y-1] != runeStr2[x-1] {
				i = 1
			}
			column[y] = utils.Min(
				utils.Min(column[y]+1, // insert
					column[y-1]+1), // delete
				lastkey+i) // substitution
			lastkey = oldkey
		}
	}

	return column[runeStr1len]
}

// OSADamerauLevenshteinDistance calculate the distance between two string
// Optimal string alignment distance variant that use extension of the Wagner-Fisher dynamic programming algorithm
// Doesn't allow multiple transformations on a same substring
// Allowing insertions, deletions, substitutions and transpositions to change one string to the second
// Compatible with non-ASCII characters
func OSADamerauLevenshteinDistance(str1, str2 string) int {
	// Convert string parameters to rune arrays to be compatible with non-ASCII
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	// Get and store length of these strings
	runeStr1len := len(runeStr1)
	runeStr2len := len(runeStr2)
	if runeStr1len == 0 {
		return runeStr2len
	} else if runeStr2len == 0 {
		return runeStr1len
	} else if utils.Equal(runeStr1, runeStr2) {
		return 0
	}

	// 2D Array
	matrix := make([][]int, runeStr1len+1)
	for i := 0; i <= runeStr1len; i++ {
		matrix[i] = make([]int, runeStr2len+1)
		for j := 0; j <= runeStr2len; j++ {
			matrix[i][j] = 0
		}
	}

	for i := 0; i <= runeStr1len; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= runeStr2len; j++ {
		matrix[0][j] = j
	}

	var count int
	for i := 1; i <= runeStr1len; i++ {
		for j := 1; j <= runeStr2len; j++ {
			if runeStr1[i-1] == runeStr2[j-1] {
				count = 0
			} else {
				count = 1
			}

			matrix[i][j] = utils.Min(utils.Min(matrix[i-1][j]+1, matrix[i][j-1]+1), matrix[i-1][j-1]+count) // insertion, deletion, substitution
			if i > 1 && j > 1 && runeStr1[i-1] == runeStr2[j-2] && runeStr1[i-2] == runeStr2[j-1] {
				matrix[i][j] = utils.Min(matrix[i][j], matrix[i-2][j-2]+1) // translation
			}
		}
	}
	return matrix[runeStr1len][runeStr2len]
}

// DamerauLevenshteinDistance calculate the distance between two string
// This algorithm computes the true Damerau–Levenshtein distance with adjacent transpositions
// Allowing insertions, deletions, substitutions and transpositions to change one string to the second
// Compatible with non-ASCII characters
func DamerauLevenshteinDistance(str1, str2 string) int {
	// Convert string parameters to rune arrays to be compatible with non-ASCII
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	// Get and store length of these strings
	runeStr1len := len(runeStr1)
	runeStr2len := len(runeStr2)
	if runeStr1len == 0 {
		return runeStr2len
	} else if runeStr2len == 0 {
		return runeStr1len
	} else if utils.Equal(runeStr1, runeStr2) {
		return 0
	}

	// Create alphabet based on input strings
	da := make(map[rune]int)
	for i := 0; i < runeStr1len; i++ {
		da[runeStr1[i]] = 0
	}
	for i := 0; i < runeStr2len; i++ {
		da[runeStr2[i]] = 0
	}

	// 2D Array for distance matrix : matrix[0..str1.length+2][0..s2.length+2]
	matrix := make([][]int, runeStr1len+2)
	for i := 0; i <= runeStr1len+1; i++ {
		matrix[i] = make([]int, runeStr2len+2)
		for j := 0; j <= runeStr2len+1; j++ {
			matrix[i][j] = 0
		}
	}

	// Maximum possible distance
	maxDist := runeStr1len + runeStr2len

	// Initialize matrix
	matrix[0][0] = maxDist
	for i := 0; i <= runeStr1len; i++ {
		matrix[i+1][0] = maxDist
		matrix[i+1][1] = i
	}
	for i := 0; i <= runeStr2len; i++ {
		matrix[0][i+1] = maxDist
		matrix[1][i+1] = i
	}

	// Process edit distance
	var cost int
	for i := 1; i <= runeStr1len; i++ {
		db := 0
		for j := 1; j <= runeStr2len; j++ {
			i1 := da[runeStr2[j-1]]
			j1 := db
			if runeStr1[i-1] == runeStr2[j-1] {
				cost = 0
				db = j
			} else {
				cost = 1
			}

			matrix[i+1][j+1] = utils.Min(
				utils.Min(
					matrix[i+1][j]+1,  // Addition
					matrix[i][j+1]+1), // Deletion
				utils.Min(
					matrix[i][j]+cost, // Substitution
					matrix[i1][j1]+(i-i1-1)+1+(j-j1-1))) // Transposition
		}

		da[runeStr1[i-1]] = i
	}

	return matrix[runeStr1len+1][runeStr2len+1]
}
