package helpers

import (
	"log"
)

// --- helpers ---
// TODO: why do they not work in context
func Must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func Must0(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
