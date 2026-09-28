//go:build linux || darwin

package installer

import (
	"reflect"
	"testing"
)

func TestExtras(t *testing.T) {
	byBin := map[string]*pkgManagerDef{}
	for i := range packageManagers {
		byBin[packageManagers[i].bin] = &packageManagers[i]
	}

	tests := []struct {
		pm   string
		want []string
	}{
		{pmApt, []string{"php8.3-curl", "php8.3-mbstring", "php8.3-xml", "php8.3-zip"}},
		{pmDnf, []string{"php8.3-php-mbstring", "php8.3-php-xml", "php8.3-php-pecl-zip"}},
		{pmYum, []string{"php8.3-php-mbstring", "php8.3-php-xml", "php8.3-php-pecl-zip"}},
		{pmPacman, nil},
		{pmZypper, nil},
	}
	for _, tt := range tests {
		if got := extras(byBin[tt.pm], "8.3"); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("extras(%s) = %v, want %v", tt.pm, got, tt.want)
		}
	}
}
