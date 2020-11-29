package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/hanwen/go-fuse/v2/fuse"
)

const (
	InodeRoot uint64 = 1 + iota
	InodeDir1
	InodeFile1
)

type iInfo struct {
	attr     fuse.Attr
	dir      bool
	children []fuse.DirEntry
}

var (
	currentTime = uint64(time.Now().Unix())
	// 1Gi bytes
	builtinContent = make([]byte, 1<<30)
	inodeTree      map[uint64]iInfo
)

func init() {
	rand.Read(builtinContent)

	inodeTree = map[uint64]iInfo{
		InodeRoot: {
			attr: fuse.Attr{
				Ino:   InodeRoot,
				Size:  1,
				Nlink: 1,
				Mode:  0775 | fuse.S_IFDIR,
				Atime: currentTime,
				Mtime: currentTime,
				Ctime: currentTime,
			},
			dir: true,
			children: []fuse.DirEntry{
				{
					Ino:  InodeDir1,
					Name: "dir-1",
					Mode: 0775 | fuse.S_IFDIR,
				},
			},
		},
		InodeDir1: {
			attr: fuse.Attr{
				Ino:   InodeDir1,
				Size:  1,
				Nlink: 1,
				Mode:  0775 | fuse.S_IFDIR,
				Atime: currentTime,
				Mtime: currentTime,
				Ctime: currentTime,
			},
			dir: true,
			children: []fuse.DirEntry{
				{
					Ino:  InodeFile1,
					Name: "file-1",
					Mode: 0775 | fuse.S_IFREG,
				},
			},
		},
		InodeFile1: {
			attr: fuse.Attr{
				Ino:   InodeFile1,
				Size:  uint64(len(builtinContent)),
				Mode:  0444 | fuse.S_IFREG,
				Atime: currentTime,
				Mtime: currentTime,
				Ctime: currentTime,
			},
		},
	}
}

func findInode(name string, children []fuse.DirEntry) (uint64, fuse.Status) {
	for _, child := range children {
		if child.Name == name {
			return child.Ino, fuse.OK
		}
	}

	return 0, fuse.ENOENT
}

type HanwenFS struct {
	fuse.RawFileSystem
}

func NewHanwenFS() fuse.RawFileSystem {
	return &HanwenFS{
		RawFileSystem: fuse.NewDefaultRawFileSystem(),
	}
}

func (fs *HanwenFS) StatFs(cancel <-chan struct{}, input *fuse.InHeader, out *fuse.StatfsOut) fuse.Status {
	fmt.Printf("StatFS input=%+v\n", input)
	return fuse.OK
}

func (fs *HanwenFS) Init(svr *fuse.Server) {
	fmt.Printf("Init server=%+v kernelSettings=%+v\n", svr, svr.KernelSettings())
}

func (fs *HanwenFS) Lookup(cancel <-chan struct{}, header *fuse.InHeader, name string, out *fuse.EntryOut) fuse.Status {
	// fmt.Printf("Lookup header=%+v name=%s\n", header, name)
	parent, ok := inodeTree[header.NodeId]
	if !ok {
		return fuse.ENOENT
	}

	childInode, err := findInode(name, parent.children)
	if err != fuse.OK {
		return err
	}

	out.NodeId = childInode
	out.Attr = inodeTree[childInode].attr
	if out.Ino == 0 {
		out.Ino = out.NodeId
	}
	// fmt.Printf("Lookup out.inode=%d out=%+v\n", out.Ino, out)
	return fuse.OK
}

func (fs *HanwenFS) GetAttr(cancel <-chan struct{}, input *fuse.GetAttrIn, out *fuse.AttrOut) fuse.Status {
	// fmt.Printf("GetAttr input=%+v\n", input)
	info, ok := inodeTree[input.NodeId]
	if !ok {
		return fuse.ENOENT
	}
	out.Attr = info.attr
	return fuse.OK
}

func (fs *HanwenFS) OpenDir(cancel <-chan struct{}, input *fuse.OpenIn, out *fuse.OpenOut) fuse.Status {
	// fmt.Printf("OpenDir input=%+v\n", input)
	return fuse.OK
}

func (fs *HanwenFS) ReadDir(cancel <-chan struct{}, input *fuse.ReadIn, out *fuse.DirEntryList) fuse.Status {
	// fmt.Printf("ReadDir input=%+v\n", input)
	info, ok := inodeTree[input.NodeId]
	if !ok {
		return fuse.ENOENT
	}
	if !info.dir {
		return fuse.EIO
	}

	for _, child := range info.children {
		out.AddDirEntry(child)
	}

	return fuse.OK
}

func (fs *HanwenFS) ReadDirPlus(cancel <-chan struct{}, input *fuse.ReadIn, out *fuse.DirEntryList) fuse.Status {
	// fmt.Printf("ReadDirPlus input=%+v\n", input)
	info, ok := inodeTree[input.NodeId]
	if !ok {
		return fuse.ENOENT
	}
	if !info.dir {
		return fuse.EIO
	}

	if input.Offset > uint64(len(info.children)) {
		return fuse.OK
	}

	children := info.children[input.Offset:]
	for _, child := range children {
		out.AddDirLookupEntry(child)
		// if entryDest == nil {
		// 	break
		// }
		// fs.Lookup(cancel, &input.InHeader, child.Name, entryDest)
	}

	return fuse.OK
}

func (fs *HanwenFS) Open(cancel <-chan struct{}, input *fuse.OpenIn, out *fuse.OpenOut) fuse.Status {
	// fmt.Printf("Open input=%+v\n", input)
	return fuse.OK
}

func (fs *HanwenFS) Read(cancel <-chan struct{}, input *fuse.ReadIn, buf []byte) (fuse.ReadResult, fuse.Status) {
	// fmt.Printf("Read input=%+v\n", input)
	contentSize := uint64(len(builtinContent))
	if input.Offset >= contentSize {
		return fuse.ReadResultData(nil), fuse.OK
	}
	end := uint64(len(buf)) + input.Offset
	if end > contentSize {
		return fuse.ReadResultData(builtinContent[input.Offset:]), fuse.OK
	}

	return fuse.ReadResultData(builtinContent[input.Offset:end]), fuse.OK
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("must be specify mount path\n")
		return
	}
	svr, err := fuse.NewServer(NewHanwenFS(), os.Args[1], &fuse.MountOptions{
		FsName:                   "hanwen",
		Debug:                    false,
		DirectMount:              true,
		IgnoreSecurityLabels:     true,
		Options:                  []string{"ro"},
		ExplicitDataCacheControl: true,
	})
	if err != nil {
		panic(err)
	}
	svr.Serve()
}
