package version

import (
	"errors"
	"testing"
)

type fakeResolver struct {
	v   string
	err error
}

func (f fakeResolver) ResolveLTS() (string, error) { return f.v, f.err }

func TestIsAlias(t *testing.T) {
	for in, want := range map[string]bool{"lts": true, "LTS": true, "Lts": true, "8.3": false, "": false, "latest": false} {
		if got := IsAlias(in); got != want {
			t.Errorf("IsAlias(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Run("concrete version passes through without calling resolver", func(t *testing.T) {
		got, wasAlias, err := Resolve("8.2", fakeResolver{err: errors.New("must not be called")})
		if err != nil || wasAlias || got != "8.2" {
			t.Fatalf("Resolve = (%q, %v, %v), want (\"8.2\", false, nil)", got, wasAlias, err)
		}
	})

	t.Run("lts resolves via resolver", func(t *testing.T) {
		got, wasAlias, err := Resolve("lts", fakeResolver{v: "8.4"})
		if err != nil || !wasAlias || got != "8.4" {
			t.Fatalf("Resolve = (%q, %v, %v), want (\"8.4\", true, nil)", got, wasAlias, err)
		}
	})

	t.Run("resolver error is wrapped", func(t *testing.T) {
		boom := errors.New("network down")
		_, wasAlias, err := Resolve("lts", fakeResolver{err: boom})
		if !errors.Is(err, boom) || !wasAlias {
			t.Fatalf("Resolve error = %v, wasAlias = %v; want wrapped %v, true", err, wasAlias, boom)
		}
	})
}
