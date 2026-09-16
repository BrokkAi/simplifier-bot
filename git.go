package simplifierbot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BrokkAi/simplifier-bot/internal/osrun"
)

type checkout struct{ config Config }

func (g checkout) git(ctx context.Context, args ...string) (string, error) {
	return osrun.Run(ctx, g.config.Directory, map[string]string{"GIT_TERMINAL_PROMPT": "0"}, append([]string{"git"}, args...)...)
}
func (g checkout) head(ctx context.Context) (string, error) {
	return g.git(ctx, "rev-parse", "refs/remotes/origin/"+g.config.Branch)
}
func (g checkout) open(ctx context.Context) error {
	if _, err := os.Stat(g.config.Directory); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(g.config.Directory), 0700); err != nil {
			return err
		}
		if _, err := osrun.Run(ctx, "", map[string]string{"GIT_TERMINAL_PROMPT": "0"}, "git", "clone", "--branch", g.config.Branch, "--", g.config.Remote, g.config.Directory); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	root, err := g.git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	if root != g.config.Directory {
		return errors.New("directory must be the root of a managed clone")
	}
	remote, err := g.git(ctx, "remote", "get-url", "origin")
	if err != nil {
		return err
	}
	if remote != g.config.Remote {
		return errors.New("managed clone origin differs from configuration")
	}
	if _, err := g.git(ctx, "check-ref-format", "refs/heads/"+g.config.Branch); err != nil {
		return err
	}
	_, err = g.git(ctx, "fetch", "--prune", "origin", "+refs/heads/"+g.config.Branch+":refs/remotes/origin/"+g.config.Branch)
	return err
}

// itemWorktree provides an immutable checkout at one exact revision. Issue
// assessment uses the branch head; PR assessment uses GitHub's pull-request ref.
func (g checkout) itemWorktree(ctx context.Context, name, ref, expected string) (checkout, string, error) {
	if _, err := g.git(ctx, "fetch", "--", ref); err != nil {
		return checkout{}, "", err
	}
	rev, err := g.git(ctx, "rev-parse", "FETCH_HEAD")
	if err != nil {
		return checkout{}, "", err
	}
	if rev != expected {
		return checkout{}, "", fmt.Errorf("GitHub revision moved: expected %s, fetched %s", expected, rev)
	}
	dir := filepath.Join(g.config.Directory+"-items", name)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
			return checkout{}, "", err
		}
		if _, err := g.git(ctx, "worktree", "add", "--detach", "--", dir, rev); err != nil {
			return checkout{}, "", err
		}
	} else if err != nil {
		return checkout{}, "", err
	}
	w := g
	w.config.Directory = dir
	if head, verifyErr := w.git(ctx, "rev-parse", "HEAD"); verifyErr == nil && head != rev {
		if _, diffErr := w.git(ctx, "diff", "--exit-code", "HEAD", "--"); diffErr == nil {
			if _, err := g.git(ctx, "worktree", "remove", "--", dir); err != nil {
				return checkout{}, "", err
			}
			return g.itemWorktree(ctx, name, ref, expected)
		}
	}
	return w, rev, w.verify(ctx, rev)
}
func (g checkout) verify(ctx context.Context, expected string) error {
	root, err := g.git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	if root != g.config.Directory {
		return errors.New("assessment directory is not its worktree root")
	}
	head, err := g.git(ctx, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != expected {
		return errors.New("agent changed the assessment commit")
	}
	if _, err := g.git(ctx, "diff", "--exit-code", "HEAD", "--"); err != nil {
		return fmt.Errorf("assessment modified tracked source: %w", err)
	}
	return nil
}
