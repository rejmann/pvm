package php

import (
	"reflect"
	"testing"
)

func TestLatestPatchPerBranch(t *testing.T) {
	all := MajorResponse{
		"8.3.9":  {},
		"8.3.10": {},
		"8.3.2":  {},
		"8.2.30": {},
		"8.2.4":  {},
		"8.4.0":  {},
		"8.1":    {}, // no patch component: ignored
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

func TestSplitVersion(t *testing.T) {
	tests := map[string][]string{
		"8.3.10": {"8", "3", "10"},
		"8.3":    {"8", "3"},
		"8":      {"8"},
		"":       {""},
	}
	for in, want := range tests {
		if got := splitVersion(in); !reflect.DeepEqual(got, want) {
			t.Errorf("splitVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVersionKey(t *testing.T) {
	tests := []struct {
		in   string
		want [3]int
	}{
		{"8.3.10", [3]int{8, 3, 10}},
		{"8.3", [3]int{8, 3, 0}},
		{"8.3.0RC1", [3]int{8, 3, 1}},
		{"1.2.3.4", [3]int{1, 2, 3}},
	}
	for _, tt := range tests {
		if got := versionKey(splitVersion(tt.in)); got != tt.want {
			t.Errorf("versionKey(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestBranchKeyOrdering(t *testing.T) {
	if !(branchKey("7.4") < branchKey("8.0") && branchKey("8.0") < branchKey("8.10")) {
		t.Errorf("branchKey ordering wrong: 7.4=%d 8.0=%d 8.10=%d",
			branchKey("7.4"), branchKey("8.0"), branchKey("8.10"))
	}
	if got := branchKey("8"); got != 0 {
		t.Errorf("branchKey(\"8\") = %d, want 0", got)
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
