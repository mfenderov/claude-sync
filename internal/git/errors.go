package git

import "fmt"

// GitOperationError represents a generic git operation failure
type GitOperationError struct {
	Op     string // Operation name (e.g., "pull", "push", "commit")
	Path   string // Repository path
	Err    error  // Underlying error
	Output string // Command output if available
}

var _ error = &GitOperationError{}

func (e *GitOperationError) Error() string {
	if e.Output != "" {
		return fmt.Sprintf("git %s failed for %s: %v\nOutput: %s", e.Op, e.Path, e.Err, e.Output)
	}
	return fmt.Sprintf("git %s failed for %s: %v", e.Op, e.Path, e.Err)
}

func (e *GitOperationError) Unwrap() error {
	return e.Err
}

// ConflictError represents a merge conflict
type ConflictError struct {
	Path  string   // Repository path
	Files []string // Files with conflicts
}

var _ error = &ConflictError{}

func (e *ConflictError) Error() string {
	if len(e.Files) > 0 {
		return fmt.Sprintf("merge conflicts in %s: %v", e.Path, e.Files)
	}
	return fmt.Sprintf("merge conflicts detected in %s", e.Path)
}

// RemoteError represents a remote repository access error
type RemoteError struct {
	URL string // Remote URL
	Op  string // Operation (validate, fetch, push)
	Err error  // Underlying error
}

var _ error = &RemoteError{}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("remote %s failed for %s: %v", e.Op, e.URL, e.Err)
}

func (e *RemoteError) Unwrap() error {
	return e.Err
}

// DirectoryError represents a directory-related error
type DirectoryError struct {
	Path string // Directory path
	Op   string // Operation (e.g., "access", "create")
	Err  error  // Underlying error
}

var _ error = &DirectoryError{}

func (e *DirectoryError) Error() string {
	return fmt.Sprintf("directory %s failed for %s: %v", e.Op, e.Path, e.Err)
}

func (e *DirectoryError) Unwrap() error {
	return e.Err
}
