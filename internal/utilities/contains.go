package utilities

// Contains checks if a string is present in a slice of strings.
func Contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
