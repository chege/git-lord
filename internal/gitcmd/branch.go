package gitcmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BranchInfo represents metadata about a single branch.
type BranchInfo struct {
	Name        string
	IsRemote    bool
	IsHead      bool
	LastCommit  time.Time
	LastAuthor  string
	LastSubject string
	CommitCount int
}

// ListBranches returns all branches with metadata using git for-each-ref.
func ListBranches(ctx context.Context) ([]BranchInfo, error) {
	// Format: refname|committerdate|authorname|contents:subject|objectname
	cmd := exec.CommandContext(ctx, "git", "for-each-ref",
		"--format=%(refname:short)|%(committerdate:iso8601)|%(authorname)|%(contents:subject)|%(objectname)",
		"refs/heads/", "refs/remotes/")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git for-each-ref failed: %w", err)
	}

	var branches []BranchInfo
	scanner := bufio.NewScanner(&stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}

		name := parts[0]
		dateStr := parts[1]
		author := parts[2]
		subject := parts[3]

		// Determine if remote
		isRemote := strings.HasPrefix(name, "origin/") || strings.Contains(name, "/")

		// Parse date
		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05 -0700", dateStr)
			if err != nil {
				t = time.Now()
			}
		}

		branches = append(branches, BranchInfo{
			Name:        name,
			IsRemote:    isRemote,
			IsHead:      false, // Will be updated below
			LastCommit:  t,
			LastAuthor:  author,
			LastSubject: subject,
			CommitCount: 0, // Will be populated separately
		})
	}

	// Get current branch to mark IsHead
	currentCmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	var currentOut bytes.Buffer
	currentCmd.Stdout = &currentOut
	_ = currentCmd.Run() // Ignore errors, branch might be in detached HEAD state
	currentBranch := strings.TrimSpace(currentOut.String())

	// Get HEAD commit for comparison
	headCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	var headOut bytes.Buffer
	headCmd.Stdout = &headOut
	headHash := ""
	if err := headCmd.Run(); err == nil {
		headHash = strings.TrimSpace(headOut.String())
	}

	// Update IsHead flags and get commit counts
	for i := range branches {
		// Check if this is the current branch
		if branches[i].Name == currentBranch {
			branches[i].IsHead = true
		}

		// Check if HEAD points to this branch
		branchCmd := exec.CommandContext(ctx, "git", "rev-parse", branches[i].Name)
		var branchOut bytes.Buffer
		branchCmd.Stdout = &branchOut
		if err := branchCmd.Run(); err == nil {
			branchHash := strings.TrimSpace(branchOut.String())
			if headHash != "" && branchHash == headHash {
				branches[i].IsHead = true
			}
		}

		// Get commit count for this branch
		countCmd := exec.CommandContext(ctx, "git", "rev-list", "--count", branches[i].Name)
		var countOut bytes.Buffer
		countCmd.Stdout = &countOut
		if err := countCmd.Run(); err == nil {
			var count int
			if _, scanErr := fmt.Sscanf(countOut.String(), "%d", &count); scanErr == nil {
				branches[i].CommitCount = count
			}
		}
	}

	return branches, nil
}

// GetDefaultBranch returns the default branch name (main, master, develop, trunk, or current).
func GetDefaultBranch(ctx context.Context) (string, error) {
	// Try symbolic-ref first
	cmd := exec.CommandContext(ctx, "git", "symbolic-ref", "refs/remotes/origin/HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err == nil {
		// Output is like "refs/remotes/origin/main"
		ref := strings.TrimSpace(stdout.String())
		parts := strings.Split(ref, "/")
		if len(parts) >= 3 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: check common names
	commonNames := []string{"main", "master", "develop", "trunk"}

	for _, name := range commonNames {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", name)
		if err := cmd.Run(); err == nil {
			return name, nil
		}
	}

	// Last resort: use current branch
	currentCmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	var currentOut bytes.Buffer
	currentCmd.Stdout = &currentOut

	if err := currentCmd.Run(); err == nil {
		branch := strings.TrimSpace(currentOut.String())
		if branch != "" {
			return branch, nil
		}
	}

	// Fallback to "main"
	return "main", nil
}

// IsBranchMerged checks if the given branch has been merged into target.
func IsBranchMerged(ctx context.Context, branch, target string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--merged", target)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("git branch --merged failed: %w", err)
	}

	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Remove leading * which indicates current branch
		if strings.HasPrefix(line, "*") {
			line = strings.TrimSpace(line[1:])
		}
		if line == branch {
			return true, nil
		}
	}

	return false, nil
}

// GetBranchAheadBehind returns the number of commits ahead and behind branch compared to target.
func GetBranchAheadBehind(ctx context.Context, branch, target string) (ahead, behind int, err error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", branch+"..."+target)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return 0, 0, fmt.Errorf("git rev-list --left-right --count failed: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	parts := strings.Split(output, "\t")
	if len(parts) >= 2 {
		_, _ = fmt.Sscanf(parts[0], "%d", &ahead)
		_, _ = fmt.Sscanf(parts[1], "%d", &behind)
	}

	return ahead, behind, nil
}

// DeleteBranch deletes a local branch.
func DeleteBranch(ctx context.Context, branch string, force bool) error {
	args := []string{"branch", "-d"}
	if force {
		args[1] = "-D" // Force delete
	}
	args = append(args, branch)

	cmd := exec.CommandContext(ctx, "git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git branch -d %s failed: %w", branch, err)
	}
	return nil
}

// DeleteRemoteBranch deletes a branch on the remote.
func DeleteRemoteBranch(ctx context.Context, remote, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "push", remote, "--delete", branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push %s --delete %s failed: %w", remote, branch, err)
	}
	return nil
}

// GetCurrentBranch returns the name of the current branch.
func GetCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git branch --show-current failed: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
