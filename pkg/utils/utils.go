package utils

// GetAnnotation returns a thing
func GetAnnotation(annotations map[string]string, a string) string {
	if res, ok := annotations[a]; ok {
		return res
	}
	return ""
}

// Contains verifies if a list of strings contains a given string
func Contains(l []string, s string) bool {
	for _, elem := range l {
		if elem == s {
			return true
		}
	}
	return false
}
