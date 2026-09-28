package version

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidVersion = errors.New("invalid version")

type Version struct {
	Major    int
	Minor    int
	Patch    int
	hasPatch bool
}

func Parse(s string) (Version, error) {
	if s == "" {
		return Version{}, fmt.Errorf("%w: empty string", ErrInvalidVersion)
	}

	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return Version{}, fmt.Errorf("%w: %q", ErrInvalidVersion, s)
	}

	nums := make([]int, len(parts))
	for i, p := range parts {
		if p == "" {
			return Version{}, fmt.Errorf("%w: %q", ErrInvalidVersion, s)
		}
		if len(p) > 1 && p[0] == '0' {
			return Version{}, fmt.Errorf("%w: leading zeros not allowed in %q", ErrInvalidVersion, s)
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("%w: %q", ErrInvalidVersion, s)
		}
		nums[i] = n
	}

	v := Version{Major: nums[0], Minor: nums[1]}
	if len(nums) == 3 {
		v.Patch = nums[2]
		v.hasPatch = true
	}
	return v, nil
}

func (v Version) Compare(other Version) int {
	for _, pair := range [][2]int{
		{v.Major, other.Major},
		{v.Minor, other.Minor},
		{v.Patch, other.Patch},
	} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	return 0
}

// HasPatch reports whether the patch component was given explicitly ("8.3.1" vs "8.3").
func (v Version) HasPatch() bool {
	return v.hasPatch
}

// Branch is the major.minor part, e.g. "8.3" for 8.3.30.
func (v Version) Branch() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor)
}

// Branch returns the major.minor part of s ("8.3.30" → "8.3"), or s itself
// when it is not a valid version.
func Branch(s string) string {
	v, err := Parse(s)
	if err != nil {
		return s
	}
	return v.Branch()
}

// Compare orders two version strings for sorting: by version, then textually
// ("8.3" before "8.3.0"). Strings that are not versions sort first.
func Compare(a, b string) int {
	va, errA := Parse(a)
	vb, errB := Parse(b)
	switch {
	case errA != nil && errB != nil:
		return strings.Compare(a, b)
	case errA != nil:
		return -1
	case errB != nil:
		return 1
	}
	if c := va.Compare(vb); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}
