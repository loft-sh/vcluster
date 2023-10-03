package random

import (
	"math/rand"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

// RandomString creates a new random string with the given length
func RandomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
