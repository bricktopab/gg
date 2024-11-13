package main

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

type ExternalGit struct {
}

func (g *ExternalGit) CreateLocalBranch(name string) {
	output, err := exec.Command("git", "checkout", "-b", name).CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to create local branch: %s", output)
	}
}

func (g *ExternalGit) GetBranchName() string {
	output, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to get current branch name: %s", output)
	}
	return strings.TrimSpace(string(output))
}

func (g *ExternalGit) CreatePR() string {
	return "-"
}

func (g *ExternalGit) OpenPR(title string) {
	output, err := exec.Command("git", "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to get remote URL: %s", output)
	}
	remoteURL := strings.TrimSpace(string(output))
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.SplitN(remoteURL, ":", 2)
		if len(parts) == 2 {
			host := strings.TrimPrefix(parts[0], "git@")
			repo := parts[1]
			remoteURL = "https://" + host + "/" + repo
		}
	}
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	encodedTitle := url.QueryEscape(title)

	branch := g.GetBranchName()

	createPRURL := fmt.Sprintf("%s/compare/main...%s?quick_pull=1&title=%s", remoteURL, branch, encodedTitle)

	switch runtime.GOOS {
	case "windows":
		err = exec.Command("cmd", "/c", "start", createPRURL).Start() // #nosec G204
	case "darwin":
		err = exec.Command("open", createPRURL).Start() // #nosec G204
	default:
		err = exec.Command("xdg-open", createPRURL).Start() // #nosec G204
	}
	if err != nil {
		log.Fatalf("Failed to open URL: %v", err)
	}
}
