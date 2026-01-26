package utils

import "strings"

func Fig() string {
	lines := []string{
		"██╗  ██╗██╗███╗   ██╗██████╗ ██████╗ ██╗ █████╗ ",
		"██║ ██╔╝██║████╗  ██║██╔══██╗██╔══██╗██║██╔══██╗",
		"█████╔╝ ██║██╔██╗ ██║██║  ██║██████╔╝██║███████║",
		"██╔═██╗ ██║██║╚██╗██║██║  ██║██╔══██╗██║██╔══██║",
		"██║  ██╗██║██║ ╚████║██████╔╝██║  ██║██║██║  ██║",
		"╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚═════╝ ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝",
	}
	/* Colors that I like
	123 + 1: Red to Purple
	159 + 1: Red to Purple but more "alive"

	*/
	start := 159
	step := 1

	var b strings.Builder
	for i, line := range lines {
		color := start + i + step
		b.WriteString("\033[38;5;")
		b.WriteString(intToString(color))
		b.WriteString("m")
		b.WriteString(line)
		b.WriteString("\033[0m\n")
	}
	return b.String()
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func ToSansBold(text string) string {
	var result []rune
	for _, r := range text {
		switch {
		case r >= 'A' && r <= 'Z':
			result = append(result, 0x1D5D4+(r-'A'))
		case r >= 'a' && r <= 'z':
			result = append(result, 0x1D5EE+(r-'a'))
		default:
			result = append(result, r)
		}
	}
	return string(result)
}
