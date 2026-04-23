package php

import (
	"context"
	"fmt"
)

const urlBase = "https://www.php.net/releases/index.php?json"

type Supported struct {
	Date              string   `json:"date"`
	SupportedVersions []string `json:"supported_versions,omitempty"`
	Museum            bool     `json:"museum"`
	Version           string   `json:"version"`
}

type BranchResponse = map[string]Supported

func FetchAllBranches(ctx context.Context) ([]Branch, error) {
	supported, err := httpRequest[BranchResponse](ctx, urlBase, HttpMethod.Get, nil)
	if err != nil {
		return nil, err
	}

	type result struct {
		branches []Branch
		err      error
	}

	var majors []string
	for name := range supported {
		majors = append(majors, name)
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
	var firstErr error
	for range majors {
		r := <-ch
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}

		if firstErr != nil {
			return nil, firstErr
		}

		all = append(all, r.branches...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no PHP releases found")
	}

	return all, nil
}

func fetchMajorBranches(
	ctx context.Context,
	major string,
	supported BranchResponse,
) ([]Branch, error) {
	url := urlBase + "&major=" + major
	supported, err := httpRequest[BranchResponse](ctx, url, HttpMethod.Get, nil)
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for name, info := range supported {
		if info.Museum {
			continue
		}

		status := StatusEOL
		if len(info.SupportedVersions) > 0 {
			status = StatusSupported
		}

		branches = append(branches, Branch{
			Name:              name,
			Latest:            info.Version,
			Status:            status,
			SupportedVersions: info.SupportedVersions,
		})
	}

	return branches, nil
}
