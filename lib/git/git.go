package git

import (
	"crypto/sha1"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"percipio.com/gopi/lib/logger"
)

type CommitInfo struct {
	Hash      string
	ShortHash string
	Timestamp time.Time
	RepoName  string
	RefName   string
}

func GetCommitInfo(useGit bool) (*CommitInfo, error) {
	if !useGit {
		return generateTimestampHash(), nil
	}

	hash, err := execGitCommand("rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get commit hash: %w", err)
	}

	remoteURL, remoteErr := execGitCommand("config", "--get", "remote.origin.url")
	if remoteErr != nil {
		logger.Error("Failed to get remote URL: %v\n", remoteErr)
		remoteURL = "unknown"
	}

	timestamp := time.Now()
	authorTime, err := execGitCommand("log", "-1", "--format=%aI")
	if err == nil {
		parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(authorTime))
		if parseErr == nil {
			timestamp = parsed
		}
	}

	return &CommitInfo{
		Hash:      strings.TrimSpace(hash),
		ShortHash: strings.TrimSpace(hash)[:8],
		RepoName:  parseRepoName(remoteURL),
		RefName:   strings.TrimSpace(remoteURL),
		Timestamp: timestamp,
	}, nil
}

func execGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func generateTimestampHash() *CommitInfo {
	now := time.Now()
	timeStr := fmt.Sprintf("%d", now.UnixNano())

	h := sha1.New()
	h.Write([]byte(timeStr))
	fullHash := fmt.Sprintf("%x", h.Sum(nil))

	return &CommitInfo{
		Hash:      fullHash,
		ShortHash: fullHash[:8],
		Timestamp: now,
	}
}

func parseRepoName(remoteURL string) string {
	remoteURL = strings.TrimSpace(remoteURL)
	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	parts := strings.Split(remoteURL, "/")
	if len(parts) >= 2 {
		return fmt.Sprintf("%s/%s", parts[len(parts)-2], parts[len(parts)-1])
	}
	return "unknown"
}
