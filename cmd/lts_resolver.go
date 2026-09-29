package cmd

import (
	"context"

	"github.com/rejmann/pvm/internal/phpnet"
)

type phpLTSResolver struct {
	ctx context.Context
}

func (r phpLTSResolver) ResolveLTS() (string, error) {
	return phpnet.LatestLTS(r.ctx)
}
