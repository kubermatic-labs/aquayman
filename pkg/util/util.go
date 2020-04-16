package util

func StringSliceContains(s []string, needle string) bool {
	for _, item := range s {
		if item == needle {
			return true
		}
	}

	return false
}
