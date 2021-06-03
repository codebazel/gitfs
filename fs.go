package gitfs

import (
	"io/fs"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// New returns a readonly fs.FS filesystem based on object.Tree.
func New(tree *object.Tree) fs.FS {
	return &gitFS{tree: tree}
}

type gitFS struct {
	tree *object.Tree
}

// Open opens the named file.
//
// When Open returns an error, it should be of type *PathError
// with the Op field set to "open", the Path field set to name,
// and the Err field describing the problem.
//
// Open should reject attempts to open names that do not satisfy
// ValidPath(name), returning a *PathError with Err set to
// ErrInvalid or ErrNotExist.
func (g *gitFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, toFSError("open", name, fs.ErrInvalid)
	}

	var entry *object.TreeEntry
	switch name {
	case ".":
		entry = &object.TreeEntry{Name: ".", Mode: filemode.Dir, Hash: g.tree.Hash}
	default:
		var err error
		entry, err = g.tree.FindEntry(name)
		if err != nil {
			return nil, toFSError("open", name, err)
		}
	}

	file, err := newGitFile(g.tree, entry)
	if err != nil {
		return nil, toFSError("open", name, err)
	}
	return file, nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (g *gitFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, toFSError("readdir", name, fs.ErrInvalid)
	}

	var tree *object.Tree
	switch name {
	case ".":
		tree = g.tree
	default:
		var err error
		tree, err = g.tree.Tree(name)
		if err != nil {
			if _, err := g.tree.FindEntry(name); err == nil {
				// regular file returns nil entries.
				return nil, nil
			}
			return nil, toFSError("readdir", name, err)
		}
	}

	entries := make([]fs.DirEntry, len(tree.Entries))
	for i, n := 0, len(tree.Entries); i < n; i++ {
		file, err := newGitFile(tree, &tree.Entries[i])
		if err != nil {
			return nil, toFSError("readdir", name, err)
		}
		entries[i] = file
	}
	return entries, nil
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (g *gitFS) Sub(dir string) (fs.FS, error) {
	tree, err := g.tree.Tree(dir)
	if err != nil {
		return nil, toFSError("sub", dir, err)
	}
	return &gitFS{tree: tree}, nil
}

func toFSError(op, name string, err error) error {
	switch err {
	case object.ErrDirectoryNotFound, object.ErrEntryNotFound, object.ErrFileNotFound:
		err = fs.ErrNotExist
	}
	return &fs.PathError{Op: op, Path: name, Err: err}
}
