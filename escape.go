package main

var (
	EscapeWindowChangePrefix = []byte("\x1B[8;")
	EscapeWindowChangeSuffix = byte('t') // We only require this so it has been hard-encoded (compare func and length)
)
