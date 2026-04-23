package php

type Status int

const (
	StatusSupported Status = iota
	StatusEOL
)

func (s Status) String() string {
	switch s {
	case StatusSupported:
		return "supported"
	default:
		return "eol"
	}
}

type Branch struct {
	Name              string // e.g. "8.4"
	Latest            string // latest version in this branch, e.g. "8.4.20"
	Status            Status
	SupportedVersions []string
	IsMuseum          bool
}

type Release = Branch
