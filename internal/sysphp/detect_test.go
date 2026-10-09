package sysphp

import (
	"reflect"
	"sync"
	"testing"
)

func TestDetect(t *testing.T) {
	versions := map[string]string{
		"/usr/bin/php8.3":       "8.3",
		"/usr/bin/php8.10":      "8.10",
		"/usr/bin/php7.4":       "7.4",
		"/usr/local/bin/php8.3": "8.3", // same version again: the first one wins
		"/usr/bin/php-broken":   "",
		"/usr/bin/php":          "8.10",
	}
	bins := []string{
		"/usr/bin/php8.3", "/usr/bin/php8.10", "/usr/bin/php7.4",
		"/usr/local/bin/php8.3", "/usr/bin/php-broken", "/usr/bin/php",
	}

	// Every query waits for all the others to have started, so the test
	// only finishes if they really run at the same time.
	var started sync.WaitGroup
	started.Add(len(bins))
	query := func(bin string) string {
		started.Done()
		started.Wait()
		return versions[bin]
	}

	got := detect(bins, query)
	want := []PHP{
		{Version: "7.4", Binary: "/usr/bin/php7.4"},
		{Version: "8.3", Binary: "/usr/bin/php8.3"},
		{Version: "8.10", Binary: "/usr/bin/php8.10"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("detect = %+v, want %+v", got, want)
	}

	if got := detect(nil, query); got != nil {
		t.Errorf("detect(nil) = %+v, want nil", got)
	}
}
