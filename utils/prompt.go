package utils

import (
	"bufio"
	"fmt"

	"strings"
)

func Prompt(reader *bufio.Reader, text string) (string, error) {
	fmt.Print(text)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
