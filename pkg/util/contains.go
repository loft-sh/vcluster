package util

func Contains(needle string, haystack []string) bool {
	for _, n := range haystack {
		if needle == n {
			return true
		}
	}
	return false
}
