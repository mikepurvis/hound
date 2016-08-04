package vcs

import (
	"bytes"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

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

	cmd = exec.Command("git", "fetch", "--prune", "--no-tags", "--depth", "1", "origin", "+" + ref + ":remotes/origin/" + ref)
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to git fetch  %s, see output below\n%sContinuing...", dir, out)
		return "", err
	}

	cmd = exec.Command("git", "reset", "--hard", "origin/" + ref)
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to set git origin for %s, see output below\n%sContinuing...", dir, out)
		return "", err
	}

	return g.HeadRev(dir)
}

func (g *GitDriver) Clone(dir string, url string, ref string) (string, error) {
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
		log.Printf("Failed to clone %s, see output below\n%sContinuing...", url, out)
		return "", err
	}

	return g.HeadRev(dir)
}

func (g *GitDriver) SpecialFiles() []string {
	return []string{
		".git",
	}
}
