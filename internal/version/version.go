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
