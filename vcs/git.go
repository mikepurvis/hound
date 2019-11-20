package vcs

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var shaPattern, _ = regexp.Compile(`[0-9a-f]{40}$`)

func init() {
	Register(newGit, "git")
}

type GitDriver struct{}

func newGit(b []byte) (Driver, error) {
	return &GitDriver{}, nil
}

func (g *GitDriver) HeadRev(dir string) (string, error) {
	cmd := exec.Command(
		"git",
		"rev-parse",
		"HEAD")
	cmd.Dir = dir
	r, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	defer r.Close()

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if _, err := io.Copy(&buf, r); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), cmd.Wait()
}

func (g *GitDriver) Pull(dir string, url string, ref string) (string, error) {
	cmd := exec.Command("git", "remote", "set-url", "origin", url)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to set git origin for %s, see output below\n%sContinuing...", dir, out)
		return "", err
	}

	if shaPattern.MatchString(ref) {
		// Reference is a SHA, and we have to unshallow our clone to find it. See:
		// https://stackoverflow.com/a/30701724/109517
		if _, err := os.Stat(dir + "/.git/shallow"); err == nil {
			// This is a shallow clone, need to unshallow it.
			cmd = exec.Command("git", "fetch", "--unshallow", "origin")
		} else {
			// This is not a shallow clone, calling --unshallow will cause an error.
			cmd = exec.Command("git", "fetch", "origin")
		}

		cmd.Dir = dir
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to fetch %s, see output below\n%sContinuing...", dir, out)
			return "", err
		}

		cmd = exec.Command("git", "reset", "--hard", ref)
		cmd.Dir = dir
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to reset %s to sha, see output below\n%sContinuing...", dir, out)
			return "", err
		}
	} else {
		// Reference is a branch or tag; we can grab just that and prune the rest
		cmd = exec.Command("git", "fetch", "--prune", "--no-tags", "--depth", "1", "origin", "+"+ref+":remotes/origin/"+ref)
		cmd.Dir = dir
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to git fetch  %s, see output below\n%sContinuing...", dir, out)
			return "", err
		}

		cmd = exec.Command("git", "reset", "--hard", "origin/"+ref)
		cmd.Dir = dir
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to set git origin for %s, see output below\n%sContinuing...", dir, out)
			return "", err
		}
	}

	return g.HeadRev(dir)
}

func (g *GitDriver) Clone(dir string, url string, ref string) (string, error) {
	if shaPattern.MatchString(ref) {
		// Reference is a SHA, and we have to do a full clone and then find that commit.
		par, rep := filepath.Split(dir)
		cmd := exec.Command(
			"git",
			"clone",
			url,
			rep)
		cmd.Dir = par
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to full clone %s, see output below\n%sContinuing...", url, out)
			return "", err
		}

		cmd = exec.Command("git", "reset", "--hard", ref)
		cmd.Dir = dir
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to reset new clone %s to sha, see output below\n%sContinuing...", dir, out)
			return "", err
		}
	} else {
		// Reference is a branch or tag; we can shallow clone just that entity.
		par, rep := filepath.Split(dir)
		cmd := exec.Command(
			"git",
			"clone",
			"--depth", "1",
			"--branch", ref,
			url,
			rep)
		cmd.Dir = par
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to shallow clone %s, see output below\n%sContinuing...", url, out)
			return "", err
		}
	}

	return g.HeadRev(dir)
}

func (g *GitDriver) SpecialFiles() []string {
	return []string{
		".git",
	}
}
