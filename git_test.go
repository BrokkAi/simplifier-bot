package simplifierbot

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrokkAi/simplifier-bot/internal/osrun"
)

// originRepo builds a local remote so checkout exercises real git plumbing
// without network access or a live agent.
func originRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		out, err := osrun.Run(context.Background(), dir, map[string]string{
			"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@example.com",
			"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@example.com",
		}, append([]string{"git"}, args...)...)
		if err != nil {
			t.Fatalf("git %s: %v", strings.Join(args, " "), err)
		}
		return out
	}
	run("init", "--initial-branch", "master")
	run("commit", "--allow-empty", "-m", "base")
	head := run("rev-parse", "HEAD")
	return dir, head
}

func testConfig(t *testing.T, remote string) Config {
	t.Helper()
	root, err := canonical(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Remote = remote
	cfg.Directory = filepath.Join(root, "checkout")
	cfg.StateDirectory = filepath.Join(root, "state")
	return cfg
}

// The branch head is already a local ref after open; fetching it as though it
// were a remote name made every issue assessment fail with exit status 128.
func TestIssueWorktreeUsesTheFetchedBranchHead(t *testing.T) {
	remote, want := originRepo(t)
	ctx := context.Background()
	base := checkout{config: testConfig(t, remote)}
	if err := base.open(ctx); err != nil {
		t.Fatal(err)
	}
	head, err := base.head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if head != want {
		t.Fatalf("head %s, want %s", head, want)
	}
	w, err := base.itemWorktree(ctx, "issue-1", head)
	if err != nil {
		t.Fatalf("issue worktree: %v", err)
	}
	if got, err := w.git(ctx, "rev-parse", "HEAD"); err != nil || got != want {
		t.Fatalf("worktree at %s (%v), want %s", got, err, want)
	}
	if again, err := base.itemWorktree(ctx, "issue-1", head); err != nil || again.config.Directory != w.config.Directory {
		t.Fatalf("reusing an existing worktree failed: %v", err)
	}
}

func TestPullRequestWorktreeFetchesTheGitHubRef(t *testing.T) {
	remote, base0 := originRepo(t)
	ctx := context.Background()
	run := func(args ...string) string {
		t.Helper()
		out, err := osrun.Run(ctx, remote, map[string]string{
			"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@example.com",
			"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@example.com",
		}, append([]string{"git"}, args...)...)
		if err != nil {
			t.Fatalf("git %s: %v", strings.Join(args, " "), err)
		}
		return out
	}
	run("commit", "--allow-empty", "-m", "pull")
	want := run("rev-parse", "HEAD")
	run("update-ref", "refs/pull/7/head", want)
	run("update-ref", "refs/heads/master", base0)

	b := checkout{config: testConfig(t, remote)}
	if err := b.open(ctx); err != nil {
		t.Fatal(err)
	}
	rev, err := b.fetchItem(ctx, "refs/pull/7/head", want)
	if err != nil {
		t.Fatalf("fetching the pull request head: %v", err)
	}
	w, err := b.itemWorktree(ctx, "pr-7", rev)
	if err != nil {
		t.Fatalf("pull request worktree: %v", err)
	}
	if got, err := w.git(ctx, "rev-parse", "HEAD"); err != nil || got != want {
		t.Fatalf("worktree at %s (%v), want %s", got, err, want)
	}
	if _, err := b.fetchItem(ctx, "refs/pull/7/head", base0); err == nil {
		t.Fatal("a moved revision was accepted")
	}
}
