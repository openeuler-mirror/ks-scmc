package internal

func uniqueInt64(s []int64) (r []int64) {
	dict := map[int64]int8{}
	for _, i := range s {
		if _, ok := dict[i]; !ok {
			dict[i] = 1
			r = append(r, i)
		}
	}
	return r
}

func uniqueString(s []string) (r []string) {
	dict := map[string]int8{}
	for _, i := range s {
		if _, ok := dict[i]; !ok {
			dict[i] = 1
			r = append(r, i)
		}
	}
	return r
}
