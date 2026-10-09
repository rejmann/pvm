package phpnet

import (
	"context"
	"fmt"
	"sort"

	"github.com/rejmann/pvm/internal/version"
	"golang.org/x/sync/errgroup"
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
	supported, err := getJSON[SupportedResponse](ctx, urlBase)
	if err != nil {
		return nil, err
	}

	var majors []string
	for name, info := range supported {
		if !info.Museum {
			majors = append(majors, name)
		}
	}

	// One request per major, all at once; the first error cancels the others.
	perMajor := make([][]Branch, len(majors))
	g, ctx := errgroup.WithContext(ctx)
	for i, major := range majors {
		g.Go(func() error {
			branches, err := fetchMajorBranches(ctx, major, supported)
			perMajor[i] = branches
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var all []Branch
	for _, branches := range perMajor {
		all = append(all, branches...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no PHP releases found")
	}

	sortBranches(all)
	return all, nil
}

// sortBranches orders branches newest to oldest (8.10, …, 8.0, 7.4, …, 5.0).
func sortBranches(branches []Branch) {
	sort.Slice(branches, func(i, j int) bool {
		a, _ := version.Parse(branches[i].Name)
		b, _ := version.Parse(branches[j].Name)
		return a.Compare(b) > 0
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
	url := urlBase + "&max=500&version=" + major
	all, err := getJSON[MajorResponse](ctx, url)
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

// latestPatchPerBranch maps each branch ("8.3") to its newest release
// ("8.3.30"). Keys that are not a full major.minor.patch are ignored.
func latestPatchPerBranch(all MajorResponse) map[string]string {
	best := map[string]version.Version{}
	result := map[string]string{}

	for s := range all {
		v, err := version.Parse(s)
		if err != nil || !v.HasPatch() {
			continue
		}
		branch := version.Branch(s)
		if cur, ok := best[branch]; !ok || v.Compare(cur) > 0 {
			best[branch] = v
			result[branch] = s
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
	b, err := version.Parse(branch)
	if err != nil {
		return "", fmt.Errorf("invalid branch %q: %w", branch, err)
	}

	url := fmt.Sprintf("%s&max=500&version=%d", urlBase, b.Major)
	all, err := getJSON[MajorResponse](ctx, url)
	if err != nil {
		return "", err
	}

	best := latestPatchPerBranch(all)
	v, ok := best[branch]
	if !ok {
		return "", fmt.Errorf("no release found for branch %s", branch)
	}
	return v, nil
}

// LatestLTS returns the newest supported branch (e.g. "8.5").
func LatestLTS(ctx context.Context) (string, error) {
	releases, err := fetchSupported(ctx)
	if err != nil {
		return "", err
	}
	return releases[0].Name, nil
}
