package git

import "fmt"

// OperationError represents a generic git operation failure
type OperationError struct {
	Err    error  // Underlying error
	Op     string // Operation name (e.g., "pull", "push", "commit")
	Path   string // Repository path
	Output string // Command output if available
}

var _ error = &OperationError{}

func (e *OperationError) Error() string {
	if e.Output != "" {
		return fmt.Sprintf("git %s failed for %s: %v\nOutput: %s", e.Op, e.Path, e.Err, e.Output)
	}
	return fmt.Sprintf("git %s failed for %s: %v", e.Op, e.Path, e.Err)
}

func (e *OperationError) Unwrap() error {
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
	Err error
	URL string
	Op  string
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
	Err  error
	Path string
	Op   string
}

var _ error = &DirectoryError{}

func (e *DirectoryError) Error() string {
	return fmt.Sprintf("directory %s failed for %s: %v", e.Op, e.Path, e.Err)
}

func (e *DirectoryError) Unwrap() error {
	return e.Err
}
