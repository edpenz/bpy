// Copyright (C) 2015  Andrew Chambers - andrewchamberss@gmail.com

package fsutil

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/htree"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func hostFileToHashTree(store bpy.CStore, path string) ([32]byte, error) {
	fin, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer fin.Close()
	fout := htree.NewWriter(store)
	if err != nil {
		return [32]byte{}, err
	}
	_, err = io.Copy(fout, fin)
	if err != nil {
		return [32]byte{}, err
	}
	return fout.Close()
}

func CpHostDirToFs(store bpy.CStore, path string) ([32]byte, error) {
	ents, err := ioutil.ReadDir(path)
	if err != nil {
		return [32]byte{}, err
	}
	dir := make(fs.DirEnts, 0, 16)
	for _, e := range ents {
		switch {
		case e.Mode().IsRegular():
			hash, err := hostFileToHashTree(store, filepath.Join(path, e.Name()))
			if err != nil {
				return [32]byte{}, err
			}
			dir = append(dir, fs.DirEnt{
				Name: e.Name(),
				Data: hash,
				Size: e.Size(),
				Mode: e.Mode(),
			})
		case e.IsDir():
			hash, err := CpHostDirToFs(store, filepath.Join(path, e.Name()))
			if err != nil {
				return [32]byte{}, err
			}
			dir = append(dir, fs.DirEnt{
				Name: e.Name(),
				Mode: e.Mode(),
				Data: hash,
			})
		}
	}
	return fs.WriteDir(store, dir)
}

func CpHashTreeToHostFile(store bpy.CStore, hash [32]byte, dst string, mode os.FileMode) error {
	f, err := htree.NewReader(store, hash)
	if err != nil {
		return err
	}
	fout, err := os.OpenFile(dst, os.O_EXCL|os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(fout, f)
	if err != nil {
		_ = fout.Close()
		return err
	}
	return fout.Close()

}

func CpFsDirToHost(store bpy.CStore, hash [32]byte, dest string, mode os.FileMode) error {
	ents, err := fs.ReadDir(store, hash)
	if err != nil {
		return err
	}
	err = os.Mkdir(dest, mode)
	if err != nil {
		return err
	}
	for _, e := range ents {
		subp := filepath.Join(dest, e.Name)
		if e.Mode.IsDir() {
			err = CpFsDirToHost(store, e.Data, subp, e.Mode)
			if err != nil {
				return err
			}
			continue
		}
		err = CpHashTreeToHostFile(store, e.Data, subp, e.Mode)
		if err != nil {
			return err
		}
	}
	return nil
}
