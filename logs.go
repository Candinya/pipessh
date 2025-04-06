package main

import "os"

func LogError(err error) {
	_, _ = os.Stderr.WriteString(err.Error() + "\n")
}

func LogPanic(err error) {
	LogError(err)
	panic(err)
}
