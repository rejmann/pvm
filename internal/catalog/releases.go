// Package catalog lists the PHP releases published on php.net.
package catalog

import (
	"context"
	"fmt"
	"slices"

	"github.com/rejmann/pvm/internal/httpx"
	"github.com/rejmann/pvm/internal/version"
)

const (
	urlBase = "https://www.php.net/releases/index.php?json"

	// maxResponseSize guards against an unexpectedly large php.net response.
	maxResponseSize = 10 << 20
)

var client = httpx.NewClient(httpx.APITimeout)

func get[T any](ctx context.Context, url string) (T, error) {
	return httpx.GetJSON[T](ctx, client, url, maxResponseSize, nil)
}

type Supported struct {
	Date              string   `json:"date"`
	SupportedVersions []string `json:"supported_versions,omitempty"`
	Museum            bool     `json:"museum"`
	Version           string   `json:"version"`
}

type SupportedResponse = map[string]Supported

func FetchAllBranches(ctx context.Context) ([]Branch, error) {
	supported, err := get[SupportedResponse](ctx, urlBase)
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

	sortBranches(all)
	return all, nil
}

// sortBranches orders branches newest to oldest (8.10, …, 8.0, 7.4, …, 5.0).
func sortBranches(branches []Branch) {
	slices.SortFunc(branches, func(a, b Branch) int {
		return version.Compare(b.Name, a.Name)
	})
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
	all, err := fetchMajor(ctx, major)
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

func fetchMajor(ctx context.Context, major string) (MajorResponse, error) {
	return get[MajorResponse](ctx, urlBase+"&max=500&version="+major)
}

// latestPatchPerBranch maps each branch to its newest release ("8.3" →
// "8.3.10"). Entries without a patch number are ignored.
func latestPatchPerBranch(all MajorResponse) map[string]string {
	best := map[string]version.Version{}
	result := map[string]string{}

	for patch := range all {
		v, err := version.Parse(patch)
		if err != nil || !v.HasPatch() {
			continue
		}

		branch := v.Branch()
		if cur, ok := best[branch]; !ok || v.Compare(cur) > 0 {
			best[branch] = v
			result[branch] = patch
		}
	}

	return result
}

func fetchSupported(ctx context.Context) ([]Branch, error) {
	all, err := FetchAllBranches(ctx)
	if err != nil {
		return nil, err
	}

	var supported []Branch
	for _, b := range all {
		if b.Status == StatusSupported {
			supported = append(supported, b)
		}
	}
	if len(supported) == 0 {
		return nil, fmt.Errorf("no supported PHP releases found")
	}

	sortBranches(supported)
	return supported, nil
}

// LatestPatch returns the latest full version for a given branch (e.g. "8.3" → "8.3.30").
func LatestPatch(ctx context.Context, branch string) (string, error) {
	v, err := version.Parse(branch)
	if err != nil {
		return "", fmt.Errorf("invalid branch %q", branch)
	}

	all, err := fetchMajor(ctx, fmt.Sprint(v.Major))
	if err != nil {
		return "", err
	}

	latest, ok := latestPatchPerBranch(all)[v.Branch()]
	if !ok {
		return "", fmt.Errorf("no release found for branch %s", branch)
	}
	return latest, nil
}

func LatestLTS(ctx context.Context) (string, error) {
	releases, err := fetchSupported(ctx)
	if err != nil {
		return "", err
	}
	return releases[0].Name, nil
}
