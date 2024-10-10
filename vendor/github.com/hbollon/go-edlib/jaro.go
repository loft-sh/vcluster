package edlib

import "github.com/hbollon/go-edlib/internal/utils"

// JaroSimilarity return a similarity index (between 0 and 1)
// It use Jaro distance algorithm and allow only transposition operation
func JaroSimilarity(str1, str2 string) float32 {
	// Convert string parameters to rune arrays to be compatible with non-ASCII
	runeStr1 := []rune(str1)
	runeStr2 := []rune(str2)

	// Get and store length of these strings
	runeStr1len := len(runeStr1)
	runeStr2len := len(runeStr2)
	if runeStr1len == 0 || runeStr2len == 0 {
		return 0.0
	} else if utils.Equal(runeStr1, runeStr2) {
		return 1.0
	}

	var match int
	// Maximum matching distance allowed
	maxDist := utils.Max(runeStr1len, runeStr2len)/2 - 1
	// Correspondence tables (1 for matching and 0 if it's not the case)
	str1Table := make([]int, runeStr1len)
	str2Table := make([]int, runeStr2len)

	// Check for matching characters in both strings
	for i := 0; i < runeStr1len; i++ {
		for j := utils.Max(0, i-maxDist); j < utils.Min(runeStr2len, i+maxDist+1); j++ {
			if runeStr1[i] == runeStr2[j] && str2Table[j] == 0 {
				str1Table[i] = 1
				str2Table[j] = 1
				match++
				break
			}
		}
	}
	if match == 0 {
		return 0.0
	}

	var t float32
	var p int
	// Check for possible translations
	for i := 0; i < runeStr1len; i++ {
		if str1Table[i] == 1 {
			for str2Table[p] == 0 {
				p++
			}
			if runeStr1[i] != runeStr2[p] {
				t++
			}
			p++
		}
	}
	t /= 2

	return (float32(match)/float32(runeStr1len) +
		float32(match)/float32(runeStr2len) +
		(float32(match)-t)/float32(match)) / 3.0
}

// JaroWinklerSimilarity return a similarity index (between 0 and 1)
// Use Jaro similarity and after look for a common prefix (length <= 4)
func JaroWinklerSimilarity(str1, str2 string) float32 {
	// Get Jaro similarity index between str1 and str2
	jaroSim := JaroSimilarity(str1, str2)

	if jaroSim != 0.0 && jaroSim != 1.0 {
		// Convert string parameters to rune arrays to be compatible with non-ASCII
		runeStr1 := []rune(str1)
		runeStr2 := []rune(str2)

		// Get and store length of these strings
		runeStr1len := len(runeStr1)
		runeStr2len := len(runeStr2)

		var prefix int

		// Find length of the common prefix
		for i := 0; i < utils.Min(runeStr1len, runeStr2len); i++ {
			if runeStr1[i] == runeStr2[i] {
				prefix++
			} else {
				break
			}
		}

		// Normalized prefix count with Winkler's constraint
		// (prefix length must be inferior or equal to 4)
		prefix = utils.Min(prefix, 4)

		// Return calculated Jaro-Winkler similarity index
		return jaroSim + 0.1*float32(prefix)*(1-jaroSim)
	}

	return jaroSim
}
