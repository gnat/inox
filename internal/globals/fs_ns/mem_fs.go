package fs_ns

//modification of https://github.com/go-git/go-billy/blob/main/memfs/storage.go (Apache 2.0 license)

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"github.com/inoxlang/inox/internal/afs"
	"github.com/inoxlang/inox/internal/core"
)

var (
	_ core.SnapshotableFilesystem = (*MemFilesystem)(nil)
)

type MemFilesystem struct {
	s         *inMemStorage
	tempCount int

	watchers     []*VirtualFilesystemWatcher
	watchersLock sync.Mutex

	closed atomic.Bool
}

func NewMemFilesystem(maxTotalStorageSize core.ByteCount) *MemFilesystem {
	return &MemFilesystem{s: newInMemoryStorage(maxTotalStorageSize)}
}

func NewMemFilesystemFromSnapshot(snapshot core.FilesystemSnapshot, maxTotalStorageSize core.ByteCount) *MemFilesystem {
	storage := newInMemoryStorageFromSnapshot(snapshot, maxTotalStorageSize)

	return &MemFilesystem{
		s: storage,
	}
}

func (fs *MemFilesystem) Chroot(path string) (billy.Filesystem, error) {
	return nil, core.ErrNotImplemented
}

func (fs *MemFilesystem) Root() string {
	panic(core.ErrNotImplemented)
}

func (fs *MemFilesystem) Absolute(path string) (string, error) {
	if core.PathFrom(path).IsAbsolute() {
		return path, nil
	}
	return "", core.ErrNotImplemented
}

func (fs *MemFilesystem) Create(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, afs.DEFAULT_CREATE_FPERM)
}

func (fs *MemFilesystem) Open(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}

func (fs *MemFilesystem) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	f, has := fs.s.Get(filename)

	if !has {
		if !IsCreate(flag) {
			return nil, os.ErrNotExist
		}

		var err error
		f, err = fs.s.New(filename, perm, flag)
		if err != nil {
			return nil, err
		}
	} else {
		if IsExclusive(flag) {
			return nil, os.ErrExist
		}

		if target, isLink := fs.resolveLink(filename, f); isLink {
			return fs.OpenFile(target, flag, perm)
		}
	}

	if f.mode.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrCannotOpenDir, filename)
	}

	return f.Duplicate(filename, perm, flag), nil
}

func (fs *MemFilesystem) resolveLink(fullpath string, f *InMemfile) (target string, isLink bool) {
	if !isSymlink(f.mode) {
		return fullpath, false
	}

	target = string(f.content.bytes)
	if !core.PathFrom(fullpath).IsAbsolute() {
		target = fs.Join(filepath.Dir(fullpath), target)
	}

	return target, true
}

func (fs *MemFilesystem) Stat(filename string) (os.FileInfo, error) {
	f, has := fs.s.Get(filename)
	if !has {
		return nil, os.ErrNotExist
	}

	fi, _ := f.Stat()

	var err error
	if target, isLink := fs.resolveLink(filename, f); isLink {
		fi, err = fs.Stat(target)
		if err != nil {
			return nil, err
		}
	}

	// the name of the file should always the name of the stated file, so we
	// overwrite the Stat returned from the storage with it, since the
	// filename may belong to a link.

	coreFileInfo := fi.(core.FileInfo)
	coreFileInfo.BaseName_ = filepath.Base(filename)
	return coreFileInfo, nil
}

func (fs *MemFilesystem) Lstat(filename string) (os.FileInfo, error) {
	f, has := fs.s.Get(filename)
	if !has {
		return nil, os.ErrNotExist
	}

	return f.Stat()
}

func (fs *MemFilesystem) ReadDir(path string) ([]os.FileInfo, error) {
	//TODO: return error if not a dir

	if f, has := fs.s.Get(path); has {
		if target, isLink := fs.resolveLink(path, f); isLink {
			return fs.ReadDir(target)
		}
	}

	var entries []os.FileInfo
	for _, f := range fs.s.Children(path) {
		fi, _ := f.Stat()
		entries = append(entries, fi)
	}

	sort.Sort(SortableFileInfo(entries))

	return entries, nil
}

func (fs *MemFilesystem) MkdirAll(path string, perm os.FileMode) error {
	_, err := fs.s.New(path, perm|os.ModeDir, 0)
	return err
}

func (fs *MemFilesystem) TempFile(dir, prefix string) (billy.File, error) {
	return util.TempFile(fs, dir, prefix)
}

func (fs *MemFilesystem) getTempFilename(dir, prefix string) string {
	fs.tempCount++
	filename := fmt.Sprintf("%s_%d_%d", prefix, fs.tempCount, time.Now().UnixNano())
	return fs.Join(dir, filename)
}

func (fs *MemFilesystem) Rename(from, to string) error {
	return fs.s.Rename(from, to)
}

func (fs *MemFilesystem) Remove(filename string) error {
	return fs.s.Remove(filename)
}

func (fs *MemFilesystem) Join(elem ...string) string {
	result := filepath.Join(elem...)
	return result
}

func (fs *MemFilesystem) Symlink(target, link string) error {
	_, err := fs.Stat(link)
	if err == nil {
		return os.ErrExist
	}

	if !os.IsNotExist(err) {
		return err
	}

	return util.WriteFile(fs, link, []byte(target), 0777|os.ModeSymlink)
}

func (fs *MemFilesystem) Readlink(link string) (string, error) {
	f, has := fs.s.Get(link)
	if !has {
		return "", os.ErrNotExist
	}

	if !isSymlink(f.mode) {
		return "", &os.PathError{
			Op:   "readlink",
			Path: link,
			Err:  fmt.Errorf("not a symlink"),
		}
	}

	return string(f.content.bytes), nil
}

// Capabilities implements the Capable interface.
func (fs *MemFilesystem) Capabilities() billy.Capability {
	return billy.WriteCapability |
		billy.ReadCapability |
		billy.ReadAndWriteCapability |
		billy.SeekCapability |
		billy.TruncateCapability
}

func (fls *MemFilesystem) Close(ctx *core.Context) error {
	if !fls.closed.CompareAndSwap(false, true) {
		return nil
	}

	//unregister the filesystem from the watched filesystems.
	watchedVirtualFilesystemsLock.Lock()
	delete(watchedVirtualFilesystems, fls)
	watchedVirtualFilesystemsLock.Unlock()

	//stop and remove all watchers
	fls.watchersLock.Lock()
	for _, watcher := range fls.watchers {
		watcher.Close()
	}
	fls.watchers = nil
	fls.watchersLock.Unlock()

	//remove all events
	fls.s.eventQueue.Clear()
	return nil
}
