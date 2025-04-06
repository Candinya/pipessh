package main

func p[T any](v T) *T {
	return &v
}
