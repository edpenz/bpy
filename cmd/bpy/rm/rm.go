package rm

import (
	"flag"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"time"
)

func Rm() {
	flag.Parse()

	cfg, err := common.GetConfig()
	if err != nil {
		common.Die("error getting config: %s\n", err)
	}

	k, err := common.GetKey(cfg)
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(cfg, &k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	epoch, err := remote.GetEpoch(c)
	if err != nil {
		common.Die("error getting current epoch: %s\n", err.Error())
	}

	for {
		store, err := common.GetCStore(cfg, &k, c)
		if err != nil {
			common.Die("error getting content store: %s\n", err.Error())
		}

		rootHash, rootVersion, ok, err := remote.GetRoot(c, &k)
		if err != nil {
			common.Die("error fetching root hash: %s\n", err.Error())
		}
		if !ok {
			common.Die("root missing\n")
		}

		ref, err := refs.GetRef(store, rootHash)
		if err != nil {
			common.Die("error fetching ref: %s\n", err.Error())
		}

		newRoot := ref.Root
		for _, toRemove := range flag.Args() {
			newRootEnt, err := fs.Remove(store, newRoot, toRemove)
			if err != nil {
				common.Die("error removing file: %s\n", err.Error())
			}
			newRoot = newRootEnt.HTree.Data
		}

		newRefHash, err := refs.PutRef(store, refs.Ref{
			CreatedAt: time.Now().Unix(),
			Root:      newRoot,
			HasPrev:   true,
			Prev:      rootHash,
		})

		err = store.Close()
		if err != nil {
			common.Die("error closing store: %s\n", err.Error())
		}

		ok, err = remote.CasRoot(c, &k, newRefHash, bpy.NextRootVersion(rootVersion), epoch)
		if err != nil {
			common.Die("error swapping root: %s\n", err.Error())
		}

		if ok {
			break
		}

	}
}
