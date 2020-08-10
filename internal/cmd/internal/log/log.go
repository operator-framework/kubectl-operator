package log

import (
	"fmt"
	"os"
	"strings"
)

func Fatal(a ...interface{}) {
	Print(a...)
	os.Exit(1)
}

func Fatalf(f string, a ...interface{}) {
	Printf(f, a...)
	os.Exit(1)
}

func Print(a ...interface{}) {
	fmt.Println(a...)
}

func Printf(f string, a ...interface{}) {
	if !strings.HasSuffix(f, "\n") {
		f += "\n"
	}
	fmt.Printf(f, a...)
}
