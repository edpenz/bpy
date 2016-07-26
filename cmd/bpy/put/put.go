package rm

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/remote"
	"encoding/hex"
	"flag"
)

func Rm() {
	tagArg := flag.String("tag", "default", "tag put rm from")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please path to remove\n")
	}

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()
	wstore, err := common.GetCStoreWriter(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	rstore, err := common.GetCStoreReader(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	tagHash, ok, err := remote.GetTag(c, *tagArg)
	if err != nil {
		common.Die("error fetching tag hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("tag '%s' does not exist\n", *tagArg)
	}

	rootHash, err := bpy.ParseHash(tagHash)
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}

	newRoot, err := fs.Remove(rstore, wstore, rootHash, flag.Args()[0])
	if err != nil {
		common.Die("error removing file: %s\n", err.Error())
	}

	err = wstore.Close()
	if err != nil {
		common.Die("error closing store: %s\n", err.Error())
	}

	err = rstore.Close()
	if err != nil {
		common.Die("error closing store: %s\n", err.Error())
	}

	ok, err = remote.CasTag(c, *tagArg, hex.EncodeToString(rootHash[:]), hex.EncodeToString(newRoot.Data[:]))
	if err != nil {
		common.Die("creating tag: %s\n", err.Error())
	}

	if !ok {
		// XXX: loop here
		common.Die("tag concurrently modified, try again\n")
	}

}
