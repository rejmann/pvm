package cmd

import (
	"context"

	"github.com/rejmann/pvm/internal/php"
)

type phpLTSResolver struct {
	ctx context.Context
}

func (r phpLTSResolver) ResolveLTS() (string, error) {
	return php.LatestLTS(r.ctx)
}
