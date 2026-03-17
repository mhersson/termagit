package git

import (
	"context"
	"strings"
)

// ListSubmodules returns the names of all configured submodules.
func (r *Repository) ListSubmodules(ctx context.Context) ([]string, error) {
	out, err := r.runGit(ctx, "submodule", "status")
	if err != nil {
		return nil, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: " HASH path (description)" or "+HASH path (description)"
		// Strip leading status character and hash
		line = strings.TrimLeft(line, " +-U")
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			names = append(names, parts[1])
		}
	}

	return names, nil
}

// ParentRepo returns the absolute path to the parent repo if the current
// working directory is inside a submodule. Returns "" if not a submodule.
func (r *Repository) ParentRepo(ctx context.Context) (string, error) {
	out, err := r.runGit(ctx, "rev-parse", "--show-superproject-working-tree")
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(out), nil
}
