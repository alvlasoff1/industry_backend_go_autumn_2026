package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}

	shift %= n
	if shift < 0 {
		shift += n
	}

	rotated := make([]rune, n)
	copy(rotated, runes[shift:])
	copy(rotated[n-shift:], runes[:shift])
	return string(rotated)
}
