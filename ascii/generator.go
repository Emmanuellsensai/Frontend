package ascii

import (
	"os"
	"strings"
)

func ReadBanner(file string) ([]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	return lines, nil
}

func BuildAsciiMap(lines []string) map[rune][]string {
	asciiMap := make(map[rune][]string)
	char := 32
	for i := 1; i < len(lines); i += 9 {
		asciiMap[rune(char)] = lines[i : i+8]
		char++
	}
	return asciiMap
}

func PrintAscii(text string, asciiMap map[rune][]string) string {
	var result strings.Builder

	text = strings.ReplaceAll(text, "\r\n", "\n")
	for i, line := range strings.Split(text, "\n") {
		if line == "" {
			if i != 0 {
				result.WriteString("\n")
			}
			continue
		}
		for row := 0; row < 8; row++ {
			for _, ch := range line {
				if art, ok := asciiMap[ch]; ok {
					result.WriteString(art[row])
				}
			}
			result.WriteString("\n")
		}
	}
	return result.String()
}
