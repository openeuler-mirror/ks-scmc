package internal

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

func countInt64(s []int64) map[int64]int64 {
	d := make(map[int64]int64)
	for _, i := range s {
		d[i]++
	}
	return d
}
