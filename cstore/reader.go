package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"container/list"
	"errors"
	"os"
	"path/filepath"
	"snappy"
	"strings"
)

type lruent struct {
	packname string
	pack     *bpack.Reader
}

type metaIndexEnt struct {
	packname string
	idx      bpack.Index
}

type Reader struct {
	store     *client9.Client
	cachepath string
	midx      []metaIndexEnt
	lru       *list.List
}

func NewReader(store *client9.Client, cachepath string) (*Reader, error) {
	dirents, err := store.Ls("/")
	if err != nil {
		return nil, err
	}
	midx := make([]metaIndexEnt, 0, 16)
	for _, dirent := range dirents {
		if strings.HasSuffix(dirent.Name, ".bpack") {
			idx, err := getAndCacheIndex(store, dirent.Name, cachepath)
			if err != nil {
				return nil, err
			}
			midxent := metaIndexEnt{
				packname: dirent.Name,
				idx:      idx,
			}
			midx = append(midx, midxent)
		}
	}
	return &Reader{
		midx:      midx,
		lru:       list.New(),
		store:     store,
		cachepath: cachepath,
	}, nil
}

var NotFound = errors.New("hash not in cstore")

func (r *Reader) Get(hash [32]byte) ([]byte, error) {
	k := string(hash[:])
	for i := range r.midx {
		_, ok := r.midx[i].idx.Search(k)
		if ok {
			packrdr, err := r.getPackReader(r.midx[i].packname, r.midx[i].idx)
			if err != nil {
				return nil, err
			}
			buf, err := packrdr.Get(k)
			if err != nil {
				return nil, err
			}
			return snappy.Decode(nil, buf)
		}
	}
	return nil, NotFound
}

func (r *Reader) getPackReader(packname string, idx bpack.Index) (*bpack.Reader, error) {
	for e := r.lru.Front(); e != nil; e = e.Next() {
		ent := e.Value.(lruent)
		if ent.packname == packname {
			r.lru.MoveToFront(e)
			return ent.pack, nil
		}
	}
	stat, err := r.store.Stat(packname)
	if err != nil {
		return nil, err
	}
	f, err := r.store.Open(packname, proto9.OREAD)
	if err != nil {
		return nil, err
	}
	pack := bpack.NewReader(f, stat.Length)
	pack.Idx = idx
	r.lru.PushFront(lruent{packname: packname, pack: pack})
	if r.lru.Len() > 5 {
		ent := r.lru.Remove(r.lru.Back()).(lruent)
		ent.pack.Close()
	}
	return pack, nil
}

func getAndCacheIndex(store *client9.Client, packname, cachepath string) (bpack.Index, error) {
	idxpath := filepath.Join(cachepath, packname+".index")

	_, err := os.Stat(idxpath)
	if err == nil {
		f, err := os.Open(idxpath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return bpack.ReadIndex(f)
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	stat, err := store.Stat(packname)
	if err != nil {
		return nil, err
	}
	f, err := store.Open(packname, proto9.OREAD)
	if err != nil {
		return nil, err
	}
	pack := bpack.NewReader(f, stat.Length)
	defer pack.Close()
	err = pack.ReadIndex()
	if err != nil {
		return nil, err
	}
	idxf, err := os.Create(idxpath)
	if err != nil {
		return nil, err
	}
	err = bpack.WriteIndex(idxf, pack.Idx)
	if err != nil {
		return nil, err
	}
	return pack.Idx, idxf.Close()
}

func (r *Reader) Close() error {
	for e := r.lru.Front(); e != nil; e = e.Next() {
		ent := e.Value.(lruent)
		err := ent.pack.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
