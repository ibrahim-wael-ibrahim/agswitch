package brand

import "fmt"

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

func Banner(version string) string {
	if version == "" {
		version = "dev"
	}
	return fmt.Sprintf("%s%s%s%s%s\n%s%s v%s%s  %sby %s · %s%s\n",
		Cyan, Bold, ASCII, Reset, Cyan,
		Bold, Name, version, Reset,
		Muted, Author, Repository, Reset,
	)
}
