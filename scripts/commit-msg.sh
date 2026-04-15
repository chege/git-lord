#!/bin/sh
# Commit message validation hook for conventional commits
# Install: ln -s ../../scripts/commit-msg.sh .git/hooks/commit-msg

COMMIT_MSG_FILE="$1"
COMMIT_MSG=$(cat "$COMMIT_MSG_FILE")

# Conventional commit pattern
# Format: type(scope): subject
# Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
PATTERN="^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z-]+\))?: .+$"

# Check if message matches conventional commit format
if ! echo "$COMMIT_MSG" | grep -qE "$PATTERN"; then
    echo ""
    echo "Error: Commit message does not follow conventional commit format."
    echo ""
    echo "Format: <type>[(optional scope)]: <description>"
    echo ""
    echo "Types:"
    echo "  feat:     A new feature"
    echo "  fix:      A bug fix"
    echo "  docs:     Documentation only changes"
    echo "  style:    Code style changes (formatting, semicolons, etc.)"
    echo "  refactor: Code refactoring"
    echo "  perf:     Performance improvements"
    echo "  test:     Adding or updating tests"
    echo "  build:    Build system changes"
    echo "  ci:       CI configuration changes"
    echo "  chore:    Other changes that don't modify src or test files"
    echo "  revert:   Reverts a previous commit"
    echo ""
    echo "Examples:"
    echo "  feat: add CSV export support"
    echo "  fix(parser): handle empty commit messages"
    echo "  docs: update README with install instructions"
    echo ""
    exit 1
fi

# Check subject line length (should be <= 100 chars)
SUBJECT_LINE=$(echo "$COMMIT_MSG" | head -n 1)
if [ ${#SUBJECT_LINE} -gt 100 ]; then
    echo ""
    echo "Error: Commit subject line is too long (${#SUBJECT_LINE} characters)."
    echo "Maximum allowed is 100 characters."
    echo ""
    exit 1
fi

# Check that subject doesn't end with a period
if echo "$SUBJECT_LINE" | grep -qE '\.$'; then
    echo ""
    echo "Error: Commit subject should not end with a period."
    echo ""
    exit 1
fi

exit 0
