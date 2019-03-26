package util

// StringSet turns an array of strings into a hash set/map
func StringSet(arr []string) map[string]string {
	indexed := make(map[string]string, len(arr))
	for _, s := range arr {
		indexed[s] = s
	}
	return indexed
}
