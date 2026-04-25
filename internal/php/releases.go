//go:build !windows

package php

import (
	"context"
	"fmt"
	"sort"
)

const urlBase = "https://www.php.net/releases/index.php?json"

type Supported struct {
	Date              string   `json:"date"`
	SupportedVersions []string `json:"supported_versions,omitempty"`
	Museum            bool     `json:"museum"`
	Version           string   `json:"version"`
}

type SupportedResponse = map[string]Supported

func FetchAllBranches(ctx context.Context) ([]Branch, error) {
	supported, err := httpRequest[SupportedResponse](ctx, urlBase, HttpMethod.Get, nil)
	if err != nil {
		return nil, err
	}

	type result struct {
		branches []Branch
		err      error
	}

	var majors []string
	for name, info := range supported {
		if !info.Museum {
			majors = append(majors, name)
		}
	}

	ch := make(chan result, len(majors))
	for _, major := range majors {
		major := major
		go func() {
			branches, err := fetchMajorBranches(ctx, major, supported)
			ch <- result{branches, err}
		}()
	}

	var all []Branch
	for range majors {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}

		all = append(all, r.branches...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no PHP releases found")
	}

	return all, nil
}

type Major struct {
	Date   string `json:"date"`
	Museum bool   `json:"museum,omitempty"`
}

type MajorResponse = map[string]Major

func fetchMajorBranches(
	ctx context.Context,
	major string,
	supported SupportedResponse,
) ([]Branch, error) {
	url := urlBase + "&max=500&version=" + major
	all, err := httpRequest[MajorResponse](ctx, url, HttpMethod.Get, nil)
	if err != nil {
		return nil, err
	}

	best := latestPatchPerBranch(all)

	activeBranches := map[string]bool{}
	for _, b := range supported[major].SupportedVersions {
		activeBranches[b] = true
	}

	var branches []Branch
	for branch, latest := range best {
		status := StatusEOL
		if activeBranches[branch] {
			status = StatusSupported
		}

		branches = append(branches, Branch{
			Name:   branch,
			Latest: latest,
			Status: status,
		})
	}

	return branches, nil
}

func latestPatchPerBranch(all MajorResponse) map[string]string {
	type patchEntry struct {
		patch string
		key   [3]int
	}
	best := map[string]patchEntry{}

	for patchStr := range all {
		parts := splitVersion(patchStr)
		if len(parts) < 3 {
			continue
		}

		branch := parts[0] + "." + parts[1]
		k := versionKey(parts)
		cur, ok := best[branch]
		if !ok || k[2] > cur.key[2] {
			best[branch] = patchEntry{patch: patchStr, key: k}
		}
	}

	result := make(map[string]string, len(best))
	for branch, entry := range best {
		result[branch] = entry.patch
	}

	return result
}

func fetchReleases(ctx context.Context) ([]Release, error) {
	all, err := FetchAllBranches(ctx)
	if err != nil {
		return nil, err
	}

	var supported []Release
	for _, b := range all {
		if b.Status == StatusSupported {
			supported = append(supported, b)
		}
	}
	if len(supported) == 0 {
		return nil, fmt.Errorf("no supported PHP releases found")
	}

	sort.Slice(supported, func(i, j int) bool {
		return branchKey(supported[i].Name) < branchKey(supported[j].Name)
	})
	return supported, nil
}

func LatestLTS(ctx context.Context) (string, error) {
	releases, err := fetchReleases(ctx)
	if err != nil {
		return "", err
	}
	return releases[len(releases)-1].Name, nil
}
