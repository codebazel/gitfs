package gitfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

func initTestGitRepo(t *testing.T, r *git.Repository, ws string) plumbing.Hash {
	for _, d := range []string{
		filepath.Join(ws, "dir1", "dir11"),
		filepath.Join(ws, "dir1", "dir12", "dir121"),
	} {
		if err := os.MkdirAll(d, 0777); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []struct {
		name    string
		content string
	}{
		{
			name:    filepath.Join(ws, "README.md"),
			content: "# README",
		},
		{
			name:    filepath.Join(ws, "dir1", "dir11", "dir11.txt"),
			content: "dir11",
		},
		{
			name:    filepath.Join(ws, "dir1", "dir12", "dir12.txt"),
			content: "dir12",
		},
		{
			name: filepath.Join(ws, "dir1", "dir12", "dir121", "empty.txt"),
		},
	} {
		if err := os.WriteFile(f.name, []byte(f.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	w, err := r.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		t.Fatal(err)
	}

	hash, err := w.Commit("files for gitfs test", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "gitfs",
			Email: "gitfs@xxx.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func TestGitFS(t *testing.T) {
	ws := t.TempDir()
	r, err := git.Init(memory.NewStorage(), osfs.New(ws))
	if err != nil {
		t.Fatal(err)
	}
	hash := initTestGitRepo(t, r, ws)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := r.CommitObject(hash)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatal(err)
	}
	gfs := New(tree)

	t.Run("fs.Stat", func(t *testing.T) {
		checkInfo := func(t *testing.T, info fs.FileInfo, name string, size int64, mod fs.FileMode, isDir bool) {
			if info.Name() != name {
				t.Errorf("expect name: %s, got: %s", name, info.Name())
			}
			if info.Size() != size {
				t.Errorf("expect size: %d, got: %d", size, info.Size())
			}
			if info.Mode() != mod {
				t.Errorf("expect mod: %v, got: %v", mod, info.Mode())
			}
			if !info.ModTime().IsZero() {
				t.Errorf("expect mod time is zero, got: %v", info.ModTime())
			}
			if info.IsDir() != isDir {
				t.Errorf("expect IsDir: %v, got: %v", isDir, info.IsDir())
			}
			if info.Sys() != nil {
				t.Errorf("expect Sys is nil, got: %v", info.Sys())
			}
		}

		t.Run("root directory", func(t *testing.T) {
			file := "."
			info, err := fs.Stat(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			checkInfo(t, info, ".", 0, fs.ModeDir|0777, true)
		})

		t.Run("sub directory", func(t *testing.T) {
			file := filepath.Join("dir1", "dir11")
			info, err := fs.Stat(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			checkInfo(t, info, "dir11", 0, fs.ModeDir|0777, true)
		})

		t.Run("regular file", func(t *testing.T) {
			file := filepath.Join("dir1", "dir11", "dir11.txt")
			info, err := fs.Stat(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			checkInfo(t, info, "dir11.txt", 5, 0644, false)
		})

		t.Run("file not exists", func(t *testing.T) {
			_, err := fs.Stat(gfs, filepath.Join("dir1", "dir11", "file_not_exits.txt"))
			if !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("expect error is: %v ,got: %v", fs.ErrNotExist, err)
			}
		})

		t.Run("invalid path", func(t *testing.T) {
			_, err := fs.Stat(gfs, "./../abc.txt")
			if !errors.Is(err, fs.ErrInvalid) {
				t.Errorf("expect error is: %v ,got: %v", fs.ErrInvalid, err)
			}
		})
	})

	t.Run("fs.ReadFile", func(t *testing.T) {
		t.Run("file in root directory", func(t *testing.T) {
			file := "README.md"
			data, err := fs.ReadFile(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "# README" {
				t.Errorf("expect data of file %s: %s, got: %s", file, "# README", string(data))
			}
		})

		t.Run("file in sub directory", func(t *testing.T) {
			file := filepath.Join("dir1", "dir11", "dir11.txt")
			data, err := fs.ReadFile(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "dir11" {
				t.Errorf("expect data of file %s: %s, got: %s", file, "dir11", string(data))
			}
		})

		t.Run("empty file", func(t *testing.T) {
			file := filepath.Join("dir1", "dir12", "dir121", "empty.txt")
			data, err := fs.ReadFile(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "" {
				t.Errorf("expect data of file %s: %s, got: %s", file, "", string(data))
			}
		})

		t.Run("root directory", func(t *testing.T) {
			file := "."
			data, err := fs.ReadFile(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "" {
				t.Errorf("expect data of file %s: %s, got: %s", file, "", string(data))
			}
		})

		t.Run("sub directory", func(t *testing.T) {
			file := filepath.Join("dir1", "dir11")
			data, err := fs.ReadFile(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "" {
				t.Errorf("expect data of file %s: %s, got: %s", file, "", string(data))
			}
		})

		t.Run("file not exists", func(t *testing.T) {
			_, err := fs.ReadFile(gfs, filepath.Join("dir1", "dir11", "file_not_exits.txt"))
			if !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("expect error is: %v ,got: %v", fs.ErrNotExist, err)
			}
		})

		t.Run("invalid path", func(t *testing.T) {
			_, err := fs.ReadFile(gfs, "./../abc.txt")
			if !errors.Is(err, fs.ErrInvalid) {
				t.Errorf("expect error is: %v ,got: %v", fs.ErrInvalid, err)
			}
		})
	})

	t.Run("fs.ReadDir", func(t *testing.T) {
		t.Run("root dir", func(t *testing.T) {
			files, err := fs.ReadDir(gfs, ".")
			if err != nil {
				t.Fatal(err)
			}
			names := make(map[string]struct{})
			for _, f := range files {
				names[f.Name()] = struct{}{}
			}
			if len(names) != 2 {
				t.Errorf("expect %d files, got: %d", 2, len(names))
			}
			for _, f := range files {
				switch f.Name() {
				case "README.md":
					if f.IsDir() {
						t.Errorf("expect regular file: %s", f.Name())
					}
					if tp := f.Type(); tp != 0 {
						t.Errorf("expect file type: %v, got: %v", 0, tp)
					}
					t.Run("info", func(t *testing.T) {
						info, err := f.Info()
						if err != nil {
							t.Fatal(err)
						}
						if name := info.Name(); name != "README.md" {
							t.Errorf("expect file name: %s, got: %s", "README.md", name)
						}
						if size := info.Size(); size != 8 {
							t.Errorf("expect file size: %d, got: %d", 8, size)
						}
						if mode := info.Mode(); mode != fs.FileMode(0644) {
							t.Errorf("expect file mode: %v, got: %v", fs.FileMode(0644), mode)
						}
						if tm := info.ModTime(); !tm.IsZero() {
							t.Errorf("expect file modify time iz zero, got: %v", tm)
						}
						if sys := info.Sys(); sys != nil {
							t.Errorf("expect file sys is nil, got: %v", sys)
						}
					})
				case "dir1":
					if !f.IsDir() {
						t.Errorf("expect directory: %s", f.Name())
					}
					if tp := f.Type(); tp != fs.ModeDir {
						t.Errorf("expect file type: %v, got: %v", fs.ModeDir, tp)
					}
					t.Run("info", func(t *testing.T) {
						info, err := f.Info()
						if err != nil {
							t.Fatal(err)
						}
						if name := info.Name(); name != "dir1" {
							t.Errorf("expect file name: %s, got: %s", "dir1", name)
						}
						if size := info.Size(); size != 0 {
							t.Errorf("expect file size: %d, got: %d", 0, size)
						}
						if mode := info.Mode(); mode != fs.FileMode(0777)|fs.ModeDir {
							t.Errorf("expect file mode: %v, got: %v", fs.FileMode(0777)|fs.ModeDir, mode)
						}
						if tm := info.ModTime(); !tm.IsZero() {
							t.Errorf("expect file modify time iz zero, got: %v", tm)
						}
						if sys := info.Sys(); sys != nil {
							t.Errorf("expect file sys is nil, got: %v", sys)
						}
					})
				default:
					t.Fatalf("got unexpected file: %s", f.Name())
				}
			}
		})

		t.Run("sub dir", func(t *testing.T) {
			files, err := fs.ReadDir(gfs, "dir1/dir12")
			if err != nil {
				t.Fatal(err)
			}
			names := make(map[string]struct{})
			for _, f := range files {
				names[f.Name()] = struct{}{}
			}
			if len(names) != 2 {
				t.Errorf("expect %d files, got: %d", 2, len(names))
			}
			for _, f := range files {
				switch f.Name() {
				case "dir12.txt":
					if f.IsDir() {
						t.Errorf("expect regular file: %s", f.Name())
					}
					if tp := f.Type(); tp != 0 {
						t.Errorf("expect file type: %v, got: %v", 0, tp)
					}
					t.Run("info", func(t *testing.T) {
						info, err := f.Info()
						if err != nil {
							t.Fatal(err)
						}
						if name := info.Name(); name != "dir12.txt" {
							t.Errorf("expect file name: %s, got: %s", "dir12.txt", name)
						}
						if size := info.Size(); size != 5 {
							t.Errorf("expect file size: %d, got: %d", 5, size)
						}
						if mode := info.Mode(); mode != fs.FileMode(0644) {
							t.Errorf("expect file mode: %v, got: %v", fs.FileMode(0644), mode)
						}
						if tm := info.ModTime(); !tm.IsZero() {
							t.Errorf("expect file modify time iz zero, got: %v", tm)
						}
						if sys := info.Sys(); sys != nil {
							t.Errorf("expect file sys is nil, got: %v", sys)
						}
					})
				case "dir121":
					if !f.IsDir() {
						t.Errorf("expect directory: %s", f.Name())
					}
					if tp := f.Type(); tp != fs.ModeDir {
						t.Errorf("expect file type: %v, got: %v", fs.ModeDir, tp)
					}
					t.Run("info", func(t *testing.T) {
						info, err := f.Info()
						if err != nil {
							t.Fatal(err)
						}
						if name := info.Name(); name != "dir121" {
							t.Errorf("expect file name: %s, got: %s", "dir121", name)
						}
						if size := info.Size(); size != 0 {
							t.Errorf("expect file size: %d, got: %d", 0, size)
						}
						if mode := info.Mode(); mode != fs.FileMode(0777)|fs.ModeDir {
							t.Errorf("expect file mode: %v, got: %v", fs.FileMode(0777)|fs.ModeDir, mode)
						}
						if tm := info.ModTime(); !tm.IsZero() {
							t.Errorf("expect file modify time iz zero, got: %v", tm)
						}
						if sys := info.Sys(); sys != nil {
							t.Errorf("expect file sys is nil, got: %v", sys)
						}
					})
				default:
					t.Fatalf("got unexpected file: %s", f.Name())
				}
			}
		})

		t.Run("regular file", func(t *testing.T) {
			files, err := fs.ReadDir(gfs, filepath.Join("dir1", "dir11", "dir11.txt"))
			if err != nil {
				t.Errorf("expect error is nil ,got: %v", err)
			}
			if len(files) > 0 {
				t.Errorf("expect empty file list, got: %v", files)
			}
		})

		t.Run("file not exists", func(t *testing.T) {
			_, err := fs.ReadDir(gfs, filepath.Join("dir1", "dir11", "file_not_exits.txt"))
			if !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("expect error is: %v ,got: %v", fs.ErrNotExist, err)
			}
		})

		t.Run("invalid path", func(t *testing.T) {
			_, err := fs.ReadDir(gfs, "./../abc.txt")
			if !errors.Is(err, fs.ErrInvalid) {
				t.Errorf("expect error is: %v ,got: %v", fs.ErrInvalid, err)
			}
		})
	})

	t.Run("fs.Sub", func(t *testing.T) {
		t.Run("root", func(t *testing.T) {
			gfs, err := fs.Sub(gfs, ".")
			if err != nil {
				t.Fatal(err)
			}

			file := "README.md"
			data, err := fs.ReadFile(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "# README" {
				t.Errorf("expect data of file %s: %s, got: %s", file, "# README", string(data))
			}
		})

		t.Run("sub", func(t *testing.T) {
			gfs, err := fs.Sub(gfs, "dir1")
			if err != nil {
				t.Fatal(err)
			}

			file := filepath.Join("dir11", "dir11.txt")
			data, err := fs.ReadFile(gfs, file)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "dir11" {
				t.Errorf("expect data of file %s: %s, got: %s", file, "dir11", string(data))
			}
		})

		t.Run("dir not exits", func(t *testing.T) {
			_, err := fs.Sub(gfs, "dir_not_exists")
			if !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("expect error is: %v ,got: %v", fs.ErrNotExist, err)
			}
		})

		t.Run("invalid path", func(t *testing.T) {
			_, err := fs.ReadFile(gfs, "./../abc.dir")
			if !errors.Is(err, fs.ErrInvalid) {
				t.Errorf("expect error is: %v ,got: %v", fs.ErrInvalid, err)
			}
		})

		t.Run("regular file", func(t *testing.T) {
			_, err := fs.ReadFile(gfs, "README.md")
			if err != nil {
				t.Errorf("expect error is nil, got: %v", err)
			}
		})
	})
}
