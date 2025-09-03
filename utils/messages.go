package utils

import (
	"os"

	"github.com/fatih/color"
)

func ErrorPrint(format string, a ...any) {
	pfun := color.New(color.FgRed).FprintfFunc()
	pfun(os.Stderr, format, a...)
}

func WarningPrint(format string, a ...any) {
	pfun := color.New(color.FgYellow).FprintfFunc()
	pfun(os.Stderr, format, a...)
}

func SuccessPrint(format string, a ...any) {
	pfun := color.New(color.FgGreen).FprintfFunc()
	pfun(os.Stderr, format, a...)
}

func InfoPrint(format string, a ...any) {
	pfun := color.New(color.FgBlue).FprintfFunc()
	pfun(os.Stderr, format, a...)
}

func PrettyPrint(format string, a ...any) {
	pfun := color.New(color.FgHiMagenta).FprintfFunc()
	pfun(os.Stderr, format, a...)
}
