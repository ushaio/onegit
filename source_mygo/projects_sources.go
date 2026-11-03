package main

import (
	"context"
	_ "embed"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// Source marks of the code hosts Onegit recognizes. A repository without a
// chosen icon leads with the mark of the host its remote names.
//
//go:embed icons/github.svg
var githubSourceSVG []byte

//go:embed icons/gitee.svg
var giteeSourceSVG []byte

//go:embed icons/gitcode.svg
var gitcodeSourceSVG []byte

// Each mark parses once on first use.
var (
	githubSourceIcon  = sync.OnceValue(func() *ui.SVG { return ui.MustParseSVG(githubSourceSVG) })
	giteeSourceIcon   = sync.OnceValue(func() *ui.SVG { return ui.MustParseSVG(giteeSourceSVG) })
	gitcodeSourceIcon = sync.OnceValue(func() *ui.SVG { return ui.MustParseSVG(gitcodeSourceSVG) })
)

// sourceIcon is the mark of a repository's host, or nil for an unknown source.
func sourceIcon(source string) func() *ui.SVG {
	switch source {
	case "github":
		return githubSourceIcon
	case "gitee":
		return giteeSourceIcon
	case "gitcode":
		return gitcodeSourceIcon
	}
	return nil
}

// remoteHost returns the lower-case host of a Git remote URL, such as
// github.com, or an empty string when it cannot be read. It understands URL
// forms with a scheme, the scp-like git@host:owner/repo form, and a bare
// host/owner/repo spelling.
func remoteHost(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}
	if i := strings.Index(remote, "://"); i >= 0 {
		rest := remote[i+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		if end := strings.IndexAny(rest, "/:"); end >= 0 {
			rest = rest[:end]
		}
		return strings.ToLower(rest)
	}
	if at := strings.LastIndex(remote, "@"); at >= 0 {
		rest := remote[at+1:]
		if end := strings.IndexAny(rest, ":/"); end >= 0 {
			rest = rest[:end]
		}
		return strings.ToLower(rest)
	}
	if end := strings.Index(remote, "/"); end >= 0 {
		remote = remote[:end]
	}
	return strings.ToLower(remote)
}

// repositorySource maps a remote URL to the name of a source Onegit draws an
// icon for, or an empty string for any other host.
func repositorySource(remote string) string {
	switch remoteHost(remote) {
	case "github.com", "www.github.com":
		return "github"
	case "gitee.com", "www.gitee.com":
		return "gitee"
	case "gitcode.com", "www.gitcode.com", "gitcode.net", "www.gitcode.net":
		return "gitcode"
	}
	return ""
}

// readRepositorySource returns the source of a repository's preferred remote
// (origin when present, else the first) or an empty string when no remote
// names a known host. Discovery runs Git once per repository for it.
func readRepositorySource(ctx context.Context, git, path string, env []string) string {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, git, "-C", path, "remote", "-v")
	hideConsole(cmd)
	cmd.Env = env
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	origin, first := "", ""
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) != 2 || !strings.Contains(fields[1], "(fetch)") {
			continue
		}
		url := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(fields[1]), "(fetch)"))
		if fields[0] == "origin" {
			origin = url
			break
		}
		if first == "" {
			first = url
		}
	}
	if origin != "" {
		return repositorySource(origin)
	}
	return repositorySource(first)
}
