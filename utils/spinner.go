package utils

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

func NewSpinner(format string, a ...any) *spinner.Spinner {
	spin := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	spin.Suffix = fmt.Sprintf(" "+format, a...)
	spin.Color("blue")
	return spin
}
