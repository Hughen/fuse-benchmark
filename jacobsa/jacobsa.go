package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/jacobsa/fuse"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

type iInfo struct {
	attr     fuseops.InodeAttributes
	dir      bool
	children []fuseutil.Dirent
}

const (
	InodeRoot = fuseops.RootInodeID + iota
	InodeDir1
	InodeFile1
)

var (
	currentTime    = time.Now()
	builtinContent = make([]byte, 1<<30)
	inodeTree      map[fuseops.InodeID]iInfo
)

func init() {
	rand.Read(builtinContent)

	inodeTree = map[fuseops.InodeID]iInfo{
		InodeRoot: {
			attr: fuseops.InodeAttributes{
				Size:   1,
				Nlink:  1,
				Mode:   0775 | os.ModeDir,
				Atime:  currentTime,
				Mtime:  currentTime,
				Ctime:  currentTime,
				Crtime: currentTime,
			},
			dir: true,
			children: []fuseutil.Dirent{
				{
					Offset: 1,
					Inode:  InodeDir1,
					Name:   "dir-1",
					Type:   fuseutil.DT_Directory,
				},
			},
		},
		InodeDir1: {
			attr: fuseops.InodeAttributes{
				Size:   1,
				Nlink:  1,
				Mode:   0775 | os.ModeDir,
				Atime:  currentTime,
				Mtime:  currentTime,
				Ctime:  currentTime,
				Crtime: currentTime,
			},
			dir: true,
			children: []fuseutil.Dirent{
				{
					Offset: 1,
					Inode:  InodeFile1,
					Name:   "file-1",
					Type:   fuseutil.DT_File,
				},
			},
		},
		InodeFile1: {
			attr: fuseops.InodeAttributes{
				Size:   uint64(len(builtinContent)),
				Mode:   0444,
				Atime:  currentTime,
				Mtime:  currentTime,
				Ctime:  currentTime,
				Crtime: currentTime,
			},
		},
	}
}

func findInode(name string, children []fuseutil.Dirent) (fuseops.InodeID, error) {
	for _, child := range children {
		if child.Name == name {
			return child.Inode, nil
		}
	}

	return 0, fuse.ENOENT
}

type JacobsaFS struct {
	fuseutil.NotImplementedFileSystem
}

func (fs *JacobsaFS) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return nil
}

func (fs *JacobsaFS) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	parent, ok := inodeTree[op.Parent]
	if !ok {
		return fuse.ENOENT
	}

	childInode, err := findInode(op.Name, parent.children)
	if err != nil {
		return err
	}

	op.Entry.Child = childInode
	op.Entry.Attributes = inodeTree[childInode].attr

	return nil
}

func (fs *JacobsaFS) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	info, ok := inodeTree[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	op.Attributes = info.attr
	return nil
}

func (fs *JacobsaFS) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return nil
}

func (fs *JacobsaFS) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	info, ok := inodeTree[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	if !info.dir {
		return fuse.EIO
	}

	entries := info.children
	if op.Offset > fuseops.DirOffset(len(entries)) {
		return fuse.EIO
	}

	entries = entries[op.Offset:]
	for _, entry := range entries {
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], entry)
		if n == 0 {
			break
		}
		op.BytesRead += n
	}

	return nil
}

func (fs *JacobsaFS) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	return nil
}

func (fs *JacobsaFS) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	contentSize := int64(len(builtinContent))
	if op.Offset >= contentSize {
		return nil
	}
	size := len(op.Dst)
	end := int64(size) + op.Offset
	if end > contentSize {
		copy(op.Dst, builtinContent[op.Offset:])
		op.BytesRead = int(contentSize - op.Offset)
	} else {
		copy(op.Dst, builtinContent[op.Offset:end])
		op.BytesRead = size
	}

	return nil
}

func NewJacobsaFS() fuse.Server {
	fs := &JacobsaFS{}
	return fuseutil.NewFileSystemServer(fs)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("must be specify mount path\n")
		return
	}
	point, err := fuse.Mount(os.Args[1], NewJacobsaFS(), &fuse.MountConfig{
		FSName:   "jacobsa",
		ReadOnly: true,
		// ErrorLogger: log.New(os.Stderr, "error ", 0),
		// DebugLogger: log.New(os.Stdout, "debug ", 0),
	})
	if err != nil {
		panic(err)
	}
	if err := point.Join(context.Background()); err != nil {
		fmt.Printf("failed join fuse server: %v\n", err)
		return
	}
}
