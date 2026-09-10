package maps

func Copy(dest map[string]string, src map[string]string) {
	for k, v := range src {
		dest[k] = v
	}
}
