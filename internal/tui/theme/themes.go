package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Palette struct {
	Name           string
	Normal         string
	SubtleLight    string
	SubtleDark     string
	BorderLight    string
	BorderDark     string
	HighlightLight string
	HighlightDark  string
	HomeFigStart   int
	HomeFigStep    int
}

var palettes = []Palette{
	{
		Name:           "Purple Night",
		Normal:         "#EEEEEE",
		SubtleLight:    "#D9DCCF",
		SubtleDark:     "#383838",
		BorderLight:    "#D9DCCF",
		BorderDark:     "#8a2be2",
		HighlightLight: "#874BFD",
		HighlightDark:  "#7D56F4",
		HomeFigStart:   159,
		HomeFigStep:    1,
	},
	{
		Name:           "Solarized Dark",
		Normal:         "#93A1A1",
		SubtleLight:    "#EEE8D5",
		SubtleDark:     "#586E75",
		BorderLight:    "#93A1A1",
		BorderDark:     "#2AA198",
		HighlightLight: "#B58900",
		HighlightDark:  "#B58900",
		HomeFigStart:   37,
		HomeFigStep:    1,
	},
	{
		Name:           "Gruvbox Dark",
		Normal:         "#EBDBB2",
		SubtleLight:    "#D5C4A1",
		SubtleDark:     "#504945",
		BorderLight:    "#D5C4A1",
		BorderDark:     "#D79921",
		HighlightLight: "#FE8019",
		HighlightDark:  "#FE8019",
		HomeFigStart:   172,
		HomeFigStep:    1,
	},
	{
		Name:           "Dracula",
		Normal:         "#F8F8F2",
		SubtleLight:    "#E6E6E6",
		SubtleDark:     "#6272A4",
		BorderLight:    "#BD93F9",
		BorderDark:     "#BD93F9",
		HighlightLight: "#FF79C6",
		HighlightDark:  "#FF79C6",
		HomeFigStart:   141,
		HomeFigStep:    1,
	},
	{
		Name:           "Nord",
		Normal:         "#ECEFF4",
		SubtleLight:    "#D8DEE9",
		SubtleDark:     "#4C566A",
		BorderLight:    "#81A1C1",
		BorderDark:     "#81A1C1",
		HighlightLight: "#88C0D0",
		HighlightDark:  "#88C0D0",
		HomeFigStart:   110,
		HomeFigStep:    1,
	},
	{
		Name:           "Monokai",
		Normal:         "#F8F8F2",
		SubtleLight:    "#EAEAEA",
		SubtleDark:     "#49483E",
		BorderLight:    "#A6E22E",
		BorderDark:     "#A6E22E",
		HighlightLight: "#F92672",
		HighlightDark:  "#F92672",
		HomeFigStart:   118,
		HomeFigStep:    1,
	},
	{
		Name:           "One Dark",
		Normal:         "#ABB2BF",
		SubtleLight:    "#D7DAE0",
		SubtleDark:     "#4B5263",
		BorderLight:    "#61AFEF",
		BorderDark:     "#61AFEF",
		HighlightLight: "#E5C07B",
		HighlightDark:  "#E5C07B",
		HomeFigStart:   75,
		HomeFigStep:    1,
	},
	{
		Name:           "Tokyo Night",
		Normal:         "#C0CAF5",
		SubtleLight:    "#D5DFFF",
		SubtleDark:     "#565F89",
		BorderLight:    "#7AA2F7",
		BorderDark:     "#7AA2F7",
		HighlightLight: "#BB9AF7",
		HighlightDark:  "#BB9AF7",
		HomeFigStart:   111,
		HomeFigStep:    1,
	},
	{
		Name:           "Catppuccin Mocha",
		Normal:         "#CDD6F4",
		SubtleLight:    "#E6E9EF",
		SubtleDark:     "#585B70",
		BorderLight:    "#89B4FA",
		BorderDark:     "#89B4FA",
		HighlightLight: "#F5C2E7",
		HighlightDark:  "#F5C2E7",
		HomeFigStart:   147,
		HomeFigStep:    1,
	},
	{
		Name:           "Kanagawa",
		Normal:         "#DCD7BA",
		SubtleLight:    "#EDE6C9",
		SubtleDark:     "#727169",
		BorderLight:    "#7E9CD8",
		BorderDark:     "#7E9CD8",
		HighlightLight: "#D27E99",
		HighlightDark:  "#D27E99",
		HomeFigStart:   110,
		HomeFigStep:    1,
	},
	{
		Name:           "Everforest Dark",
		Normal:         "#D3C6AA",
		SubtleLight:    "#E5D9BD",
		SubtleDark:     "#5C6A72",
		BorderLight:    "#7FBBB3",
		BorderDark:     "#7FBBB3",
		HighlightLight: "#E69875",
		HighlightDark:  "#E69875",
		HomeFigStart:   108,
		HomeFigStep:    1,
	},
	{
		Name:           "Arc Dark",
		Normal:         "#D3DAE3",
		SubtleLight:    "#E8ECF2",
		SubtleDark:     "#4A5664",
		BorderLight:    "#5294E2",
		BorderDark:     "#5294E2",
		HighlightLight: "#8C9EFF",
		HighlightDark:  "#8C9EFF",
		HomeFigStart:   75,
		HomeFigStep:    1,
	},
	{
		Name:           "Ubuntu Aubergine",
		Normal:         "#F7F7F7",
		SubtleLight:    "#E8E8E8",
		SubtleDark:     "#5E2750",
		BorderLight:    "#E95420",
		BorderDark:     "#E95420",
		HighlightLight: "#AEA79F",
		HighlightDark:  "#AEA79F",
		HomeFigStart:   166,
		HomeFigStep:    1,
	},
	{
		Name:           "Mint-Y",
		Normal:         "#E8F5E9",
		SubtleLight:    "#F1FAF2",
		SubtleDark:     "#4E5A4E",
		BorderLight:    "#66BB6A",
		BorderDark:     "#66BB6A",
		HighlightLight: "#A5D6A7",
		HighlightDark:  "#A5D6A7",
		HomeFigStart:   114,
		HomeFigStep:    1,
	},
	{
		Name:           "Pop Dark",
		Normal:         "#F3F4F5",
		SubtleLight:    "#E7EAEE",
		SubtleDark:     "#4A4E59",
		BorderLight:    "#48B9C7",
		BorderDark:     "#48B9C7",
		HighlightLight: "#F5A97F",
		HighlightDark:  "#F5A97F",
		HomeFigStart:   80,
		HomeFigStep:    1,
	},
	{
		Name:           "Breeze Dark",
		Normal:         "#EFF0F1",
		SubtleLight:    "#F7F7F7",
		SubtleDark:     "#5E6468",
		BorderLight:    "#3DAEE9",
		BorderDark:     "#3DAEE9",
		HighlightLight: "#F67400",
		HighlightDark:  "#F67400",
		HomeFigStart:   75,
		HomeFigStep:    1,
	},
	{
		Name:           "Material Ocean",
		Normal:         "#A6ACCD",
		SubtleLight:    "#C3C8DF",
		SubtleDark:     "#515772",
		BorderLight:    "#82AAFF",
		BorderDark:     "#82AAFF",
		HighlightLight: "#C792EA",
		HighlightDark:  "#C792EA",
		HomeFigStart:   111,
		HomeFigStep:    1,
	},
}

func All() []Palette {
	cp := make([]Palette, len(palettes))
	copy(cp, palettes)
	return cp
}

func Default() Palette {
	return palettes[0]
}

func ByName(name string) (Palette, bool) {
	for _, p := range palettes {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Palette{}, false
}

type savedTheme struct {
	Name string `json:"name"`
}

func SaveSelected(name string) error {
	if _, ok := ByName(name); !ok {
		return fmt.Errorf("unknown theme: %s", name)
	}
	path, err := themeConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(savedTheme{Name: name})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadSelected() (Palette, error) {
	path, err := themeConfigPath()
	if err != nil {
		return Palette{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Palette{}, err
	}
	var s savedTheme
	if err := json.Unmarshal(data, &s); err != nil {
		return Palette{}, err
	}
	p, ok := ByName(s.Name)
	if !ok {
		return Palette{}, fmt.Errorf("theme not found: %s", s.Name)
	}
	return p, nil
}

func themeConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "kindria", "theme.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "kindria", "theme.json"), nil
}
