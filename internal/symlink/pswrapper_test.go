package symlink

import (
	"strings"
	"testing"
)

func TestPowerShellWrapperAvoidsPS7OnlySyntax(t *testing.T) {
	w := powerShellWrapper(`C:\Users\o'neil\pvm.exe`, `C:\pvm\shims`)

	for _, bad := range []string{"?.", "??", "Join-String"} {
		if strings.Contains(w, bad) {
			t.Errorf("wrapper uses %q, which Windows PowerShell 5.1 cannot parse", bad)
		}
	}
	if !strings.Contains(w, `$exe = 'C:\Users\o''neil\pvm.exe'`) {
		t.Errorf("wrapper does not embed the escaped pvm path:\n%s", w)
	}
}

func TestUpsertPowerShellWrapper(t *testing.T) {
	const wrapper = "# pvm-wrapper new\n# end pvm-wrapper\n"

	tests := []struct {
		name        string
		profile     string
		want        string
		wantChanged bool
	}{
		{
			name:        "empty profile",
			profile:     "",
			want:        "\n" + wrapper,
			wantChanged: true,
		},
		{
			name:        "appends after user content",
			profile:     "Set-Alias ll ls\n",
			want:        "Set-Alias ll ls\n\n" + wrapper,
			wantChanged: true,
		},
		{
			name:        "replaces old block and keeps surrounding content",
			profile:     "before\r\n\n# pvm-wrapper old\r\nfoo ?.Source\r\n# end pvm-wrapper\r\nafter\r\n",
			want:        "before\r\n\n" + wrapper + "after\r\n",
			wantChanged: true,
		},
		{
			name:        "replaces unterminated block",
			profile:     "before\n# pvm-wrapper old\nbroken",
			want:        "before\n" + wrapper,
			wantChanged: true,
		},
		{
			name:        "already current",
			profile:     "before\n" + wrapper + "after\n",
			want:        "before\n" + wrapper + "after\n",
			wantChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := upsertPowerShellWrapper(tt.profile, wrapper)
			if got != tt.want {
				t.Errorf("upsertPowerShellWrapper =\n%q\nwant\n%q", got, tt.want)
			}
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}
