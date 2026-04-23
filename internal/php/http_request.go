package php

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const Timeout = 15 * time.Second

var httpClient = &http.Client{
	Timeout: Timeout,
}

type HttpData map[string]interface{}

var HttpMethod = struct {
	Get, Post, Put, Delete, Patch string
}{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodDelete,
	http.MethodPatch,
}

func httpRequest[T any](
	ctx context.Context,
	url string,
	method string,
	body io.Reader,
) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return zero, fmt.Errorf("creating request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, fmt.Errorf("decoding response: %w", err)
	}

	return result, nil
}
