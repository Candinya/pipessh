package main

func p[T any](v T) *T {
	return &v
}

func arrayContains[T comparable](arr []T, item T) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}
