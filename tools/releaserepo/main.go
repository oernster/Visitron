// Command releaserepo prints the repository this checkout's releases come
// from, as owner/name, read off its origin remote (Amendment 15). Every build
// script runs it (build.ps1, builddmg.sh and build_flatpak.sh) and hands the
// answer to the application through -ldflags -X, so no account is ever
// written into the source and a fork checks its own releases. It prints
// nothing when there is no git, no origin or an origin that is not on GitHub;
// the build then has no update source and the window says so.
//
// It is Go rather than a script per shell so the rule has one home: every
// platform that builds Visitron already has Go.
//
//	go run ./tools/releaserepo           the checkout's own origin
//	go run ./tools/releaserepo ADDRESS   the given remote address
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// githubRemote matches https://github.com/owner/name(.git) and
// git@github.com:owner/name(.git).
var githubRemote = regexp.MustCompile(
	`^(?:https://github\.com/|git@github\.com:)([A-Za-z0-9-]+)/([A-Za-z0-9._-]+?)(?:\.git)?/?$`)

// releaseRepo answers owner/name for a GitHub remote address, "" otherwise.
func releaseRepo(origin string) string {
	m := githubRemote.FindStringSubmatch(strings.TrimSpace(origin))
	if m == nil {
		return ""
	}
	return m[1] + "/" + m[2]
}

// origin answers the checkout's origin remote, "" when git cannot say.
func origin() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func main() {
	address := ""
	if len(os.Args) > 1 {
		address = os.Args[1]
	} else {
		address = origin()
	}
	fmt.Print(releaseRepo(address))
}
