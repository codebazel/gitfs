package gitfs

import (
	"io"
	"io/fs"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
)

func newGitFile(tree *object.Tree, entry *object.TreeEntry) (*gitFile, error) {
	mode, err := entry.Mode.ToOSFileMode()
	if err != nil {
		return nil, err
	}
	return &gitFile{tree: tree, entry: entry, mode: mode}, nil
}

type gitFile struct {
	tree  *object.Tree
	entry *object.TreeEntry
	mode  fs.FileMode

	file *object.File
	r    io.ReadCloser
}

func (g *gitFile) load() (err error) {
	if g.file != nil || g.mode.IsDir() {
		return
	}
	g.file, err = g.tree.TreeEntryFile(g.entry)
	return
}

// --- fs.File ---

func (g *gitFile) Stat() (fs.FileInfo, error) {
	if err := g.load(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *gitFile) Read(bs []byte) (int, error) {
	if g.mode.IsDir() {
		return 0, io.EOF
	}
	if err := g.load(); err != nil {
		return 0, err
	}
	if g.r == nil {
		r, err := g.file.Reader()
		if err != nil {
			return 0, err
		}
		g.r = r
	}
	return g.r.Read(bs)
}

func (g *gitFile) Close() error {
	if g.r != nil {
		return g.r.Close()
	}
	return nil
}

// --- fs.DirEntry ---

// Info returns the FileInfo for the file or subdirectory described by the entry.
// The returned FileInfo may be from the time of the original directory read
// or from the time of the call to Info. If the file has been removed or renamed
// since the directory read, Info may return an error satisfying errors.Is(err, ErrNotExist).
// If the entry denotes a symbolic link, Info reports the information about the link itself,
// not the link's target.
func (g *gitFile) Info() (fs.FileInfo, error) {
	return g.Stat()
}

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (g *gitFile) Type() fs.FileMode {
	return g.mode.Type()
}

// --- fs.FileInfo ---

func (g *gitFile) Name() string {
	return g.entry.Name
}

func (g *gitFile) Size() int64 {
	if g.file != nil {
		return g.file.Size
	}
	return 0
}

func (g *gitFile) Mode() fs.FileMode {
	return g.mode
}

func (g *gitFile) ModTime() time.Time {
	return time.Time{}
}

func (g *gitFile) IsDir() bool {
	return g.mode.IsDir()
}

func (g *gitFile) Sys() interface{} {
	return nil
}
