package php

func splitVersion(s string) []string {
	var parts []string
	cur := ""
	for _, c := range s {
		if c == '.' {
			parts = append(parts, cur)
			cur = ""

			continue
		}

		cur += string(c)
	}

	return append(parts, cur)
}

func versionKey(parts []string) [3]int {
	var k [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		k[i] = n
	}
	return k
}
