//go:build linux

package installer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rejmann/pvm/internal/sudo"
)

const (
	osReleasePath = "/etc/os-release"

	// Ondřej Surý's repository for Debian, the counterpart of the
	// ppa:ondrej/php PPA on Ubuntu.
	suryKeyringURL  = "https://packages.sury.org/debsuryorg-archive-keyring.deb"
	suryKeyringPath = "/usr/share/keyrings/debsuryorg-archive-keyring.gpg"
	suryRepoURL     = "https://packages.sury.org/php/"
	suryListPath    = "/etc/apt/sources.list.d/php.list"
)

// osRelease holds the fields of /etc/os-release pvm uses.
type osRelease struct {
	ID       string
	IDLike   []string
	Codename string
}

func (r osRelease) is(id string) bool {
	if r.ID == id {
		return true
	}
	for _, like := range r.IDLike {
		if like == id {
			return true
		}
	}
	return false
}

func readOSRelease(path string) (osRelease, error) {
	f, err := os.Open(path)
	if err != nil {
		return osRelease{}, err
	}
	defer f.Close()
	return parseOSRelease(f), nil
}

func parseOSRelease(r io.Reader) osRelease {
	vals := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		key, val, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if !ok || strings.HasPrefix(key, "#") {
			continue
		}
		vals[key] = strings.Trim(val, `"'`)
	}
	codename := vals["VERSION_CODENAME"]
	if codename == "" {
		codename = vals["DEBIAN_CODENAME"]
	}
	return osRelease{
		ID:       vals["ID"],
		IDLike:   strings.Fields(vals["ID_LIKE"]),
		Codename: codename,
	}
}

// addAptPHPRepo adds the repository that carries every PHP branch: the
// ondrej/php PPA on Ubuntu and its derivatives, packages.sury.org on Debian.
func addAptPHPRepo(stdout, stderr io.Writer) error {
	rel, err := readOSRelease(osReleasePath)
	if err != nil {
		return fmt.Errorf("detect distribution: %w", err)
	}
	switch {
	case rel.is("ubuntu"):
		return addOndrejPPA(stdout, stderr)
	case rel.is("debian"):
		return addSuryRepo(rel.Codename, stdout, stderr)
	default:
		return fmt.Errorf("pvm does not know a PHP repository for %q: add one providing php<X.Y>-cli and try again", rel.ID)
	}
}

func runAsRoot(stdout, stderr io.Writer, args ...string) error {
	cmd := sudo.Command(args...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}
	return nil
}

func addOndrejPPA(stdout, stderr io.Writer) error {
	if _, err := exec.LookPath("add-apt-repository"); err != nil {
		fmt.Fprintln(stdout, "add-apt-repository not found, installing software-properties-common...")
		if err := runAsRoot(stdout, stderr, pmApt, "install", "-y", "software-properties-common"); err != nil {
			return err
		}
	}
	fmt.Fprintln(stdout, "Package not found, adding ondrej/php PPA...")
	// add-apt-repository already runs apt-get update; a second update right
	// after it can fail on the apt lists lock.
	return runAsRoot(stdout, stderr, "add-apt-repository", "-y", "ppa:ondrej/php")
}

func addSuryRepo(codename string, stdout, stderr io.Writer) error {
	if codename == "" {
		return fmt.Errorf("cannot add packages.sury.org: no VERSION_CODENAME in %s", osReleasePath)
	}
	fmt.Fprintln(stdout, "Package not found, adding packages.sury.org PHP repository...")

	tmp, err := os.MkdirTemp("", "pvm-sury-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	keyring := tmp + "/debsuryorg-archive-keyring.deb"
	if err := downloadFile(suryKeyringURL, keyring); err != nil {
		return fmt.Errorf("download Sury keyring: %w", err)
	}
	if err := runAsRoot(stdout, stderr, "dpkg", "-i", keyring); err != nil {
		return err
	}

	list := tmp + "/php.list"
	line := suryListLine(codename)
	if err := os.WriteFile(list, []byte(line), 0o644); err != nil {
		return err
	}
	if err := runAsRoot(stdout, stderr, "install", "-m", "0644", list, suryListPath); err != nil {
		return err
	}
	return runAsRoot(stdout, stderr, pmApt, "update")
}

func suryListLine(codename string) string {
	return "deb [signed-by=" + suryKeyringPath + "] " + suryRepoURL + " " + codename + " main\n"
}

func downloadFile(url, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
