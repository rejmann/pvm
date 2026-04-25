//go:build windows

package php

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

func FetchAllBranches(_ context.Context) ([]Branch, error) {
	return fetchFromWinget()
}

func LatestLTS(_ context.Context) (string, error) {
	branches, err := fetchFromWinget()
	if err != nil {
		return "", err
	}

	var supported []Branch
	for _, b := range branches {
		if b.Status == StatusSupported {
			supported = append(supported, b)
		}
	}
	if len(supported) == 0 {
		return "", fmt.Errorf("no supported PHP releases found in winget")
	}

	sort.Slice(supported, func(i, j int) bool {
		return branchKey(supported[i].Name) < branchKey(supported[j].Name)
	})
	return supported[len(supported)-1].Name, nil
}

func fetchFromWinget() ([]Branch, error) {
	out, err := exec.Command("winget", "search", "PHP.PHP", "--source", "winget").Output()
	if err != nil {
		return nil, fmt.Errorf("winget search PHP.PHP: %w — make sure winget is available", err)
	}
	return parseWingetSearch(string(out))
}

func parseWingetSearch(output string) ([]Branch, error) {
	var branches []Branch

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)

		var id, version string
		for _, f := range fields {
			if strings.HasPrefix(f, "PHP.PHP.") {
				id = f
			}
			// version field looks like X.Y.Z and is not a PHP.* identifier
			if !strings.HasPrefix(f, "PHP") && len(strings.Split(f, ".")) == 3 {
				version = f
			}
		}

		if id == "" || version == "" {
			continue
		}

		// PHP.PHP.8.4 → ["PHP", "PHP", "8", "4"]
		parts := strings.SplitN(id, ".", 4)
		if len(parts) < 4 {
			continue
		}
		branch := parts[2] + "." + parts[3]

		major := 0
		for _, c := range parts[2] {
			if c >= '0' && c <= '9' {
				major = major*10 + int(c-'0')
			}
		}

		status := StatusEOL
		if major >= 8 {
			status = StatusSupported
		}

		branches = append(branches, Branch{
			Name:   branch,
			Latest: version,
			Status: status,
		})
	}

	if len(branches) == 0 {
		return nil, fmt.Errorf("no PHP packages found — run: winget search PHP.PHP")
	}

	sort.Slice(branches, func(i, j int) bool {
		return branchKey(branches[i].Name) < branchKey(branches[j].Name)
	})
	return branches, nil
}
