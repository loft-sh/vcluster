package emailverifier

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(emailRegexString)

// Syntax stores all information about an email Syntax
type Syntax struct {
	Username string `json:"username"`
	Domain   string `json:"domain"`
	Valid    bool   `json:"valid"`
}

// ParseAddress attempts to parse an email address and return it in the form of an Syntax
func (v *Verifier) ParseAddress(email string) Syntax {

	isAddressValid := IsAddressValid(email)
	if !isAddressValid {
		return Syntax{Valid: false}
	}

	index := strings.LastIndex(email, "@")
	username := email[:index]
	domain := strings.ToLower(email[index+1:])

	return Syntax{
		Username: username,
		Domain:   domain,
		Valid:    isAddressValid,
	}
}

// IsAddressValid checks if email address is formatted correctly by using regex
func IsAddressValid(email string) bool {
	return emailRegex.MatchString(email)
}
