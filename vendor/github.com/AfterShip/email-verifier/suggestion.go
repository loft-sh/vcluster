package emailverifier

import (
	"strings"

	"github.com/hbollon/go-edlib"
)

// SuggestDomain checks if domain has a typo and suggests a similar correct domain from metadata,
// returns a suggestion
func (v *Verifier) SuggestDomain(domain string) string {
	if domain == "" {
		return ""
	}

	domain = strings.ToLower(domain)
	sld, tld := splitDomain(domain)
	// If the domain is a valid second level domain and top level domain, do not suggest anything
	if sld != "" && tld != "" {
		if suggestionSecondLevelDomains[sld] && suggestionTopLevelDomains[tld] {
			return ""
		}

	}

	closestDomain := findClosestDomain(domain, freeDomains, domainThreshold)
	if closestDomain != "" {
		if closestDomain == domain {
			// The domain exactly matches one of the suggestion domains, no suggestion provided.
			return ""
		}
		// The domain closely matches one of the suggestion domains
		return closestDomain
	}

	var localTypo bool
	closestDomain = domain

	closestSecondLevelDomain := findClosestDomain(sld, suggestionSecondLevelDomains, secondLevelThreshold)
	closestTopLevelDomain := findClosestDomain(tld, suggestionTopLevelDomains, topLevelThreshold)

	if closestSecondLevelDomain != "" && closestSecondLevelDomain != sld {
		localTypo = true
		closestDomain = strings.Replace(closestDomain, sld, closestSecondLevelDomain, -1)

	}
	if closestTopLevelDomain != "" && closestTopLevelDomain != tld && sld != "" {
		localTypo = true
		closestDomain = strings.Replace(closestDomain, tld, closestTopLevelDomain, -1)
	}

	if localTypo {
		return closestDomain
	}

	return ""
}

// findClosestDomain finds the string most similar to the domain via Levenshtein algorithms.
func findClosestDomain(domain string, domains map[string]bool, threshold float32) string {
	var maxDist = float32(-1)
	var closestDomain string

	if domain == "" || len(domains) == 0 {
		return closestDomain
	}

	for d := range domains {
		if domain == d {
			return domain
		}

		dist, _ := edlib.StringsSimilarity(domain, d, edlib.Levenshtein)
		if dist > maxDist {
			maxDist = dist
			closestDomain = d
		}
	}

	if maxDist >= threshold && closestDomain != "" {
		return closestDomain
	}

	return ""
}
