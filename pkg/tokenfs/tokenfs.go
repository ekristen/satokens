package tokenfs

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"log"
	"net/http"
	"os"
	"sync"
)

func NewTokenFS() (fuse.Server, error) {
	fs := &TokenFS{}

	return fuseutil.NewFileSystemServer(fs), nil
}

type TokenFS struct {
	fuseutil.NotImplementedFileSystem

	mu            sync.Mutex
	tokenContents []byte // GUARDED_BY(mu)
}

const (
	rootInode fuseops.InodeID = fuseops.RootInodeID + iota
	tokenInode
)

//--------------------------------------------------------------------------------------------------------------

func (fs *TokenFS) rootAttributes() fuseops.InodeAttributes {
	return fuseops.InodeAttributes{
		Nlink: 1,
		Mode:  0777 | os.ModeDir,
	}
}

// LOCKS_REQUIRED(fs.mu)
func (fs *TokenFS) tokenAttributes() fuseops.InodeAttributes {
	return fuseops.InodeAttributes{
		Nlink: 1,
		Mode:  0777,
		Size:  uint64(len(fs.tokenContents)),
	}
}

// LOCKS_REQUIRED(fs.mu)
func (fs *TokenFS) getAttributes(id fuseops.InodeID) (fuseops.InodeAttributes, error) {
	switch id {
	case fuseops.RootInodeID:
		return fs.rootAttributes(), nil
	case tokenInode:
		return fs.tokenAttributes(), nil
	default:
		return fuseops.InodeAttributes{}, fuse.ENOENT
	}
}

//--------------------------------------------------------------------------------------------------------------

func (fs *TokenFS) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	return nil
}

func (fs *TokenFS) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check.
	if op.Parent != fuseops.RootInodeID {
		return fuse.ENOENT
	}

	// Set up the entry.
	switch op.Name {
	case "token":
		op.Entry = fuseops.ChildInodeEntry{
			Child:      tokenInode,
			Attributes: fs.tokenAttributes(),
		}
		if err := fs.ReadRemoteFile(); err != nil {
			return err
		}
	default:
		return fuse.ENOENT
	}

	return nil
}

func (fs *TokenFS) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var err error
	op.Attributes, err = fs.getAttributes(op.Inode)
	return err
}

func (fs *TokenFS) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Ignore any changes and simply return existing attributes.
	var err error
	op.Attributes, err = fs.getAttributes(op.Inode)
	return err
}

func (fs *TokenFS) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check.
	if op.Inode != tokenInode {
		return fuse.ENOSYS
	}

	if err := fs.ReadRemoteFile(); err != nil {
		return err
	}

	return nil
}

func (fs *TokenFS) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.ReadRemoteFile(); err != nil {
		return err
	}

	// Ensure the offset is in range.
	if op.Offset > int64(len(fs.tokenContents)) {
		return nil
	}

	// Read what we can.
	op.BytesRead = copy(op.Dst, fs.tokenContents[op.Offset:])

	return nil
}

func (fs *TokenFS) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Sanity check.
	switch op.Inode {
	case fuseops.RootInodeID:
		if err := fs.ReadRemoteFile(); err != nil {
			return err
		}
	default:
		return fuse.ENOENT
	}

	return nil
}

func (fs *TokenFS) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Create the appropriate listing.
	var dirEntries []fuseutil.Dirent

	switch op.Inode {
	case fuseops.RootInodeID:
		dirEntries = []fuseutil.Dirent{
			fuseutil.Dirent{
				Offset: 1,
				Inode:  tokenInode,
				Name:   "token",
				Type:   fuseutil.DT_File,
			},
		}

	default:
		return fmt.Errorf("unexpected inode: %v", op.Inode)
	}

	// If the offset is for the end of the listing, we're done. Otherwise we
	// expect it to be for the start.
	switch op.Offset {
	case fuseops.DirOffset(len(dirEntries)):
		return nil

	case 0:

	default:
		return fmt.Errorf("unexpected offset: %v", op.Offset)
	}

	// Fill in the listing.
	for _, de := range dirEntries {
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], de)

		// We don't support doing this in anything more than one shot.
		if n == 0 {
			return fmt.Errorf("couldn't fit listing in %v bytes", len(op.Dst))
		}

		op.BytesRead += n
	}

	return nil
}

type Payload struct {
	Contents []byte `json:"contents"`
}

func (fs *TokenFS) ReadRemoteFile() error {
	resp, err := http.Get("http://localhost:44044")
	if err != nil {
		log.Fatalln(err)
	}

	var data Payload
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	fs.tokenContents = data.Contents

	return nil
}
