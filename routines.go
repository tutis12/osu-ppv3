package main

import (
	"fmt"
	"os"
	"runtime"
)

func Run(f func()) {
	go func() {
		defer Recover()
		f()
	}()
}

func Recover() {
	if r := recover(); r != nil {
		HandlePanic(r)
	}
}

func HandlePanic(panic any) {
	defer os.Exit(1)

	buf := make([]byte, 100000)
	n := runtime.Stack(buf, false)
	buf = buf[:n]

	fmt.Printf("Panic: %v\n\n%s\n\n", panic, string(buf))
}

func PanicF(format string, a ...any) {
	panic(fmt.Sprintf(format, a...))
}
