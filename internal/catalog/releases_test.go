package catalog

import (
	"reflect"
	"testing"
)

func TestLatestPatchPerBranch(t *testing.T) {
	all := MajorResponse{
		"8.3.9":    {},
		"8.3.10":   {},
		"8.3.2":    {},
		"8.2.30":   {},
		"8.2.4":    {},
		"8.4.0":    {},
		"8.1":      {}, // no patch component: ignored
		"8.0.0RC1": {}, // not a release: ignored
	}

	got := latestPatchPerBranch(all)
	want := map[string]string{
		"8.2": "8.2.30",
		"8.3": "8.3.10",
		"8.4": "8.4.0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("latestPatchPerBranch = %v, want %v", got, want)
	}
}

func TestSortBranches(t *testing.T) {
	branches := []Branch{{Name: "8.3"}, {Name: "5.6"}, {Name: "8.10"}, {Name: "7.4"}, {Name: "8.0"}}
	sortBranches(branches)

	var got []string
	for _, b := range branches {
		got = append(got, b.Name)
	}
	want := []string{"8.10", "8.3", "8.0", "7.4", "5.6"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortBranches = %v, want %v", got, want)
	}
}

func TestStatusString(t *testing.T) {
	if StatusSupported.String() != "supported" || StatusEOL.String() != "eol" {
		t.Errorf("Status.String: got %q / %q", StatusSupported, StatusEOL)
	}
}
