package version

import (
	"fmt"
	"strings"
)

type Resolver interface {
	ResolveLTS() (string, error)
}

func IsAlias(s string) bool {
	return strings.EqualFold(s, "lts")
}

func Resolve(s string, r Resolver) (concrete string, wasAlias bool, err error) {
	if !IsAlias(s) {
		return s, false, nil
	}

	v, err := r.ResolveLTS()
	if err != nil {
		return "", true, fmt.Errorf("could not resolve lts version: %w", err)
	}

	return v, true, nil
}
