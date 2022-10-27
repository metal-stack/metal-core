package switcher

func contains(l []string, e string) bool {
	for _, i := range l {
		if i == e {
			return true
		}
	}
	return false
}
