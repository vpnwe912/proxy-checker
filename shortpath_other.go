//go:build !windows

package main

// ToShortPath is a no-op on non-Windows systems.
func ToShortPath(path string) string {
	return path
}
