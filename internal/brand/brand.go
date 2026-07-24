package brand

import (
	"fmt"
	"strings"
)

const (
	Name       = "agswitch"
	Author     = "Ibrahim Wael"
	GitHubUser = "ibrahim-wael-ibrahim"
	Repository = "github.com/ibrahim-wael-ibrahim/agswitch"

	Reset   = "\033[0m"
	Muted   = "\033[2m"
	Bold    = "\033[1m"
	Cyan    = "\033[1;36m"
	Magenta = "\033[1;35m"
	Green   = "\033[1;32m"
	Yellow  = "\033[1;33m"
	Red     = "\033[1;31m"
	White   = "\033[1;37m"
)

const ASCII = `   _    ____ ____        _ _       _
  / \  / ___/ ___|__  __(_) |_ ___| |__
 / _ \| |  _\___ \\ \/ / | __/ __| '_ \
/ ___ \ |_| |___) |>  <| | || (__| | | |
/_/   \_\____|____//_/\_\_|\__\___|_| |_|
`

// VersionLabel returns a human-readable version with exactly one leading v.
// Development builds remain "dev" instead of becoming "vdev".
func VersionLabel(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || strings.EqualFold(version, "dev") {
		return "dev"
	}
	version = strings.TrimLeft(version, "vV")
	if version == "" {
		return "dev"
	}
	return "v" + version
}

func Banner(version string) string {
	return fmt.Sprintf("%s%s%s%s\n%s%s %s%s  %sby %s · %s%s\n",
		Bold, ASCII, Reset,
		Bold, Name, VersionLabel(version), Reset,
		Muted, Author, Repository, Reset,
	)
}
