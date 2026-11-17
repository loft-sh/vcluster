package stringutil

func Merge(haystack []string, haystack2 []string) []string {
	ret := append([]string{}, haystack...)
	ret = append(ret, haystack2...)
	return RemoveDuplicates(ret)
}

func Contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func RemoveDuplicates(arr []string) []string {
	newArr := []string{}
	for _, v := range arr {
		if !Contains(newArr, v) {
			newArr = append(newArr, v)
		}
	}
	return newArr
}
