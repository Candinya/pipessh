package main

import "os"

func LogError(err error) {
	_, _ = os.Stderr.WriteString(err.Error() + "\r\n")
}

func LogPanic(err error) {
	LogError(err)
	os.Exit(1)
}
