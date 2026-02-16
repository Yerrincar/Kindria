package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Fig() string {
	return FigWithPalette(159, 1)
}

func figLines() []string {
	return []string{
		"██╗  ██╗██╗███╗   ██╗██████╗ ██████╗ ██╗ █████╗ ",
		"██║ ██╔╝██║████╗  ██║██╔══██╗██╔══██╗██║██╔══██╗",
		"█████╔╝ ██║██╔██╗ ██║██║  ██║██████╔╝██║███████║",
		"██╔═██╗ ██║██║╚██╗██║██║  ██║██╔══██╗██║██╔══██║",
		"██║  ██╗██║██║ ╚████║██████╔╝██║  ██║██║██║  ██║",
		"╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚═════╝ ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝",
	}
}

func FigWithColor(colorHex string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorHex)).
		Render(strings.Join(figLines(), "\n"))
}

func FigWithGradient(startHex, endHex string) string {
	lines := figLines()
	if len(lines) == 0 {
		return ""
	}
	sr, sg, sb, okStart := parseHexColor(startHex)
	er, eg, eb, okEnd := parseHexColor(endHex)
	if !okStart || !okEnd {
		return FigWithColor(startHex)
	}

	var b strings.Builder
	den := len(lines) - 1
	if den <= 0 {
		return FigWithColor(startHex)
	}
	for i, line := range lines {
		t := float64(i) / float64(den)
		r := lerp(sr, er, t)
		g := lerp(sg, eg, t)
		bl := lerp(sb, eb, t)
		colorHex := fmt.Sprintf("#%02x%02x%02x", r, g, bl)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex)).Render(line))
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func parseHexColor(hex string) (int, int, int, bool) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, false
	}
	r, err := strconv.ParseInt(hex[1:3], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	g, err := strconv.ParseInt(hex[3:5], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	b, err := strconv.ParseInt(hex[5:7], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(r), int(g), int(b), true
}

func lerp(a, c int, t float64) int {
	return int(float64(a) + (float64(c)-float64(a))*t)
}

func FigWithPalette(start, step int) string {
	lines := figLines()
	/* Colors that I like
	123 + 1: Red to Purple
	159 + 1: Red to Purple but more "alive"

	*/
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

func GetStarRating(val float64) string {
	const maxStars = 5
	rounded := math.Round(val*4) / 4
	if rounded < 0 {
		rounded = 0
	}
	if rounded > maxStars {
		rounded = maxStars
	}

	starStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true)

	var stars strings.Builder
	fullStars := int(math.Floor(rounded))
	fraction := rounded - float64(fullStars)
	partial := ""

	switch fraction {
	case 0.25:
		partial = "¼"
	case 0.5:
		partial = "½"
	case 0.75:
		partial = "¾"
	}

	stars.WriteString(strings.Repeat("⭐", fullStars))
	if partial != "" {
		stars.WriteString(partial)
	}

	return starStyle.Render(stars.String())
}
