package edlib

// SorensenDiceCoefficient computes the Sorensen-Dice coefficient between two strings
// Takes two strings as parameters, a split length which defines the k-gram shingle length
func SorensenDiceCoefficient(str1, str2 string, splitLength int) float32 {
	if str1 == "" && str2 == "" {
		return 0
	}
	shingle1 := Shingle(str1, splitLength)
	shingle2 := Shingle(str2, splitLength)

	intersection := float32(0)
	for i := range shingle1 {
		if _, ok := shingle2[i]; ok {
			intersection++
		}
	}
	return 2.0 * intersection / float32(len(shingle1)+len(shingle2))
}
