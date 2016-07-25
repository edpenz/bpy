package fs

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/htree"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type DirEnts []DirEnt

type DirEnt struct {
	EntName    string
	EntSize    int64
	EntModTime int64
	EntMode    os.FileMode
	Data       [32]byte
}

func (ent *DirEnt) Name() string       { return ent.EntName }
func (ent *DirEnt) Size() int64        { return ent.EntSize }
func (ent *DirEnt) Mode() os.FileMode  { return ent.EntMode }
func (ent *DirEnt) ModTime() time.Time { return time.Unix(ent.EntModTime, 0) }
func (ent *DirEnt) IsDir() bool        { return ent.EntMode.IsDir() }
func (ent *DirEnt) Sys() interface{}   { return nil }

func (dir DirEnts) Len() int           { return len(dir) }
func (dir DirEnts) Less(i, j int) bool { return dir[i].EntName < dir[j].EntName }
func (dir DirEnts) Swap(i, j int)      { dir[i], dir[j] = dir[j], dir[i] }

func WriteDir(store bpy.CStoreWriter, indir DirEnts, mode os.FileMode) ([32]byte, error) {
	var numbytes [8]byte
	var dirBuf [256]DirEnt
	var dir DirEnts

	// Best effort at stack allocating this slice
	// XXX todo benchmark the affect of this.
	// XXX should probably factor code so it doesn't need to do this copy
	if len(indir)+1 < len(dirBuf) {
		dir = dirBuf[0 : len(indir)+1]
	} else {
		dir = make(DirEnts, len(indir)+1, len(indir)+1)
	}
	copy(dir[1:], indir)
	mode |= os.ModeDir
	dir[0] = DirEnt{EntName: "", EntMode: mode}

	sort.Sort(dir)
	for i := 0; i < len(dir)-1; i++ {
		if dir[i].EntName == dir[i+1].EntName {
			return [32]byte{}, fmt.Errorf("duplicate directory entry '%s'", dir[i].EntName)
		}
		if dir[i].EntName == "." {
			return [32]byte{}, fmt.Errorf("cannot name file or folder '.'")
		}
	}
	dir[0].EntName = "."

	nbytes := 0
	for i := range dir {
		nbytes += 2 + len(dir[i].EntName) + 8 + 4 + 8 + 32
	}
	buf := bytes.NewBuffer(make([]byte, 0, nbytes))
	for _, e := range dir {
		// err is always nil for buf writes, no need to check.
		if len(e.EntName) > 65535 {
			return [32]byte{}, fmt.Errorf("directory entry name '%s' too long", e.EntName)
		}
		binary.LittleEndian.PutUint16(numbytes[0:2], uint16(len(e.EntName)))
		buf.Write(numbytes[0:2])
		buf.WriteString(e.EntName)

		binary.LittleEndian.PutUint64(numbytes[0:8], uint64(e.EntSize))
		buf.Write(numbytes[0:8])

		binary.LittleEndian.PutUint32(numbytes[0:4], uint32(e.EntMode))
		buf.Write(numbytes[0:4])

		binary.LittleEndian.PutUint64(numbytes[0:8], uint64(e.EntModTime))
		buf.Write(numbytes[0:8])

		buf.Write(e.Data[:])
	}

	tw := htree.NewWriter(store)
	_, err := tw.Write(buf.Bytes())
	if err != nil {
		return [32]byte{}, err
	}

	return tw.Close()
}

func ReadDir(store bpy.CStoreReader, hash [32]byte) (DirEnts, error) {
	var dir DirEnts
	rdr, err := htree.NewReader(store, hash)
	if err != nil {
		return nil, err
	}
	dirdata, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	for len(dirdata) != 0 {
		var hash [32]byte
		namelen := int(binary.LittleEndian.Uint16(dirdata[0:2]))
		dirdata = dirdata[2:]
		name := string(dirdata[0:namelen])
		dirdata = dirdata[namelen:]
		size := int64(binary.LittleEndian.Uint64(dirdata[0:8]))
		dirdata = dirdata[8:]
		mode := os.FileMode(binary.LittleEndian.Uint32(dirdata[0:4]))
		dirdata = dirdata[4:]
		modtime := int64(binary.LittleEndian.Uint64(dirdata[0:8]))
		dirdata = dirdata[8:]
		copy(hash[:], dirdata[0:32])
		dirdata = dirdata[32:]
		dir = append(dir, DirEnt{
			EntName:    name,
			EntSize:    size,
			EntMode:    mode,
			EntModTime: modtime,
			Data:       hash,
		})
	}
	// fill in the hash for "."
	dir[0].Data = hash
	return dir, nil
}

func Walk(store bpy.CStoreReader, hash [32]byte, fpath string) (DirEnt, error) {
	var result DirEnt
	var end int

	if fpath == "" || fpath[0] != '/' {
		fpath = "/" + fpath
	}
	fpath = path.Clean(fpath)
	pathelems := strings.Split(fpath, "/")
	if pathelems[len(pathelems)-1] == "" {
		end = len(pathelems) - 1
	} else {
		end = len(pathelems)
	}
	for i := 0; i < end; i++ {
		entname := pathelems[i]
		if entname == "" {
			entname = "."
		}
		ents, err := ReadDir(store, hash)
		if err != nil {
			return result, err
		}
		found := false
		j := 0
		for j = range ents {
			if ents[j].EntName == entname {
				found = true
				break
			}
		}
		if !found {
			return result, fmt.Errorf("no such directory: %s", entname)
		}
		if i != end-1 {
			if !ents[j].EntMode.IsDir() {
				return result, fmt.Errorf("not a directory: %s", ents[j].EntName)
			}
			hash = ents[j].Data
		} else {
			result = ents[j]
		}
	}
	if result.EntName == "." {
		result.Data = hash
	}
	return result, nil
}

type FileReader struct {
	offset uint64
	fsize  int64
	rdr    *htree.Reader
}

func (r *FileReader) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		o, err := r.rdr.Seek(uint64(off))
		r.offset = o
		return int64(o), err
	case io.SeekCurrent:
		o, err := r.rdr.Seek(r.offset + uint64(off))
		r.offset = o
		return int64(o), err
	case io.SeekEnd:
		o, err := r.rdr.Seek(uint64(r.fsize + off))
		r.offset = o
		return int64(o), err
	default:
		return int64(r.offset), fmt.Errorf("bad whence %d", whence)
	}
}

func (r *FileReader) Read(buf []byte) (int, error) {
	nread, err := r.rdr.Read(buf)
	r.offset += uint64(nread)
	return nread, err
}

func (r *FileReader) ReadAt(buf []byte, off int64) (int, error) {
	if r.offset != uint64(off) {
		_, err := r.Seek(off, io.SeekStart)
		if err != nil {
			return 0, err
		}
	}
	return io.ReadFull(r, buf)
}

func (r *FileReader) Close() error {
	// nothing to do but having Close in the api isn't bad
	// if we need to add it.
	return nil
}

func Open(store bpy.CStoreReader, roothash [32]byte, fpath string) (*FileReader, error) {
	dirent, err := Walk(store, roothash, fpath)
	if err != nil {
		return nil, err
	}
	if dirent.EntMode.IsDir() {
		return nil, fmt.Errorf("%s is a directory", fpath)
	}
	rdr, err := htree.NewReader(store, dirent.Data)
	if err != nil {
		return nil, err
	}
	return &FileReader{
		offset: 0,
		fsize:  dirent.EntSize,
		rdr:    rdr,
	}, nil
}

func Ls(store bpy.CStoreReader, roothash [32]byte, fpath string) (DirEnts, error) {
	dirent, err := Walk(store, roothash, fpath)
	if err != nil {
		return nil, err
	}
	if !dirent.EntMode.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", fpath)
	}
	ents, err := ReadDir(store, dirent.Data)
	if err != nil {
		return nil, err
	}
	return ents, nil
}

/*
func Insert(store bpy.CStoreReader, dest [32]byte, destPath string, ent DirEnt) ([32]byte, error) {
	if destPath == "" || destPath[0] != '/' {
		destPath = "/" + destPath
	}
	destPath = path.Clean(destPath)
	pathElems := strings.Split(destPath, "/")
	if pathElems[len(pathElems)-1] == "" {
		pathElems = pathElems[:len(pathElems)-1]
	}
	return insert(store, dest, pathElems, ent)
}

func insert(store bpy.CStoreReader, dest [32]byte, destPath []string, ent DirEnt) ([32]byte, error) {
	destEnts, err := ReadDir(store, dest)
	if err != nil {
		return [32]byte{}, err
	}

	if len(destPath) == 0 {
		mode := destEnts[0].EntMode
		// Reuse . entry for new entry
		destEnts[0] = ent
		return WriteDir(store, destEnts[1:], mode)
	}

	for i := 0; i < len(destEnts); i++ {
		if destEnts[i].Name == destPath[0] {
			if !destEnts[i].IsDir() {
				return [32]byte{}, fmt.Errorf("%s is not a directory", destEnts[i].EntName)
			}
			newData, err := insert(destEnts[i].Data, destPath[1:], ent)
			if err != nil {
				return [32]byte{}, err
			}
			destEnts[i].Data = newData
			return WriteDir(store, destEnts[1:], destEnts[0].EntMode)
		}
	}
	return [32]byte{}, fmt.Errorf("no folder or file named", destPath[0])
}
*/
