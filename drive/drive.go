package drive

import (
	"encoding/json"
	"errors"
	"github.com/boltdb/bolt"
	"github.com/buppyio/bpy"
	"time"
)

type PackListing struct {
	Name string
	Size uint64
	Date time.Time
}

type packState struct {
	UploadComplete bool
	Epoch          string
	Listing        PackListing
}

const (
	MetaDataBucketName = "metadata"
	PacksBucketName    = "packs"
)

var (
	ErrGCOccurred      = errors.New("concurrent garbage collection, operation failed")
	ErrGCNotRunning    = errors.New("garbage collection not running")
	ErrDuplicatePack   = errors.New("duplicate pack")
	ErrInvalidPackName = errors.New("invalid pack name")
)

type Drive struct {
	dbPath string
}

func openBoltDB(dbPath string) (*bolt.DB, error) {
	return bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
}

func nextEpoch(metaDataBucket *bolt.Bucket) (string, error) {
	epoch := bpy.NextEpoch(string(metaDataBucket.Get([]byte("epoch"))))
	err := metaDataBucket.Put([]byte("epoch"), []byte(epoch))
	if err != nil {
		return "", err
	}
	return epoch, nil
}

func Open(dbPath string) (*Drive, error) {
	db, err := openBoltDB(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket, err := tx.CreateBucketIfNotExists([]byte(MetaDataBucketName))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(PacksBucketName))
		if err != nil {
			return err
		}

		if metaDataBucket.Get([]byte("epoch")) == nil {
			epoch, err := bpy.NewEpoch()
			if err != nil {
				return err
			}

			err = metaDataBucket.Put([]byte("epoch"), []byte(epoch))
			if err != nil {
				return err
			}
		}

		if metaDataBucket.Get([]byte("gcrunning")) == nil {
			err = metaDataBucket.Put([]byte("gcrunning"), []byte("0"))
			if err != nil {
				return err
			}
		}

		if metaDataBucket.Get([]byte("rootversion")) == nil {
			ver, err := bpy.NewRootVersion()
			if err != nil {
				return err
			}

			err = metaDataBucket.Put([]byte("rootversion"), []byte(ver))
			if err != nil {
				return err
			}
		}

		if metaDataBucket.Get([]byte("rootsignature")) == nil {
			err = metaDataBucket.Put([]byte("rootsignature"), []byte(""))
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &Drive{
		dbPath: dbPath,
	}, nil
}

func (d *Drive) Close() error {
	return nil
}

func (d *Drive) Attach(keyId string) (bool, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var ok bool
	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		currentKeyId := metaDataBucket.Get([]byte("keyid"))
		if currentKeyId != nil {
			if string(currentKeyId) != keyId {
				return nil
			}
		} else {
			err = metaDataBucket.Put([]byte("keyid"), []byte(keyId))
			if err != nil {
				return err
			}
		}
		ok = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (d *Drive) GetEpoch() (string, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var epoch string

	err = db.View(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		epoch = string(metaDataBucket.Get([]byte("epoch")))
		return nil
	})

	if err != nil {
		return "", err
	}

	return epoch, nil
}

func (d *Drive) StartGC() (string, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var epoch string

	err = db.Update(func(tx *bolt.Tx) error {
		packsBucket := tx.Bucket([]byte(PacksBucketName))
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))

		epoch, err = nextEpoch(metaDataBucket)
		if err != nil {
			return err
		}

		err = metaDataBucket.Put([]byte("gcrunning"), []byte("1"))
		if err != nil {
			return err
		}

		toDelete := [][]byte{}

		err = packsBucket.ForEach(func(k, v []byte) error {
			var state packState
			err := json.Unmarshal(v, &state)
			if err != nil {
				return err
			}
			if !state.UploadComplete {
				toDelete = append(toDelete, k)
			}
			return nil
		})
		if err != nil {
			return err
		}

		for _, packName := range toDelete {
			err := packsBucket.Delete(packName)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	err = db.Close()
	if err != nil {
		return "", err
	}

	return epoch, nil
}

func (d *Drive) StopGC() error {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))

		_, err = nextEpoch(metaDataBucket)
		if err != nil {
			return err
		}

		err = metaDataBucket.Put([]byte("gcrunning"), []byte("0"))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (d *Drive) CasRoot(root, version, signature, epoch string) (bool, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var ok bool

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		rootVersion := string(metaDataBucket.Get([]byte("rootversion")))

		if bpy.NextRootVersion(rootVersion) != version {
			return nil
		}

		curEpoch := string(metaDataBucket.Get([]byte("epoch")))

		if curEpoch != epoch {
			return nil
		}

		err = metaDataBucket.Put([]byte("rootversion"), []byte(version))
		if err != nil {
			return err
		}
		err = metaDataBucket.Put([]byte("rootval"), []byte(root))
		if err != nil {
			return err
		}
		err = metaDataBucket.Put([]byte("rootsignature"), []byte(signature))
		if err != nil {
			return err
		}

		ok = true
		return nil
	})

	if err != nil {
		return false, err
	}

	err = db.Close()
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (d *Drive) GetRoot() (string, string, string, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return "", "", "", err
	}
	defer db.Close()

	var root, rootVersion, signature string

	err = db.View(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		root = string(metaDataBucket.Get([]byte("rootval")))
		signature = string(metaDataBucket.Get([]byte("rootsignature")))
		rootVersion = string(metaDataBucket.Get([]byte("rootversion")))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", "", "", err
	}

	return root, rootVersion, signature, nil
}

func (d *Drive) StartUpload(packName string) error {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(packName) > 1024 {
		return errors.New("invalid pack name")
	}

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		packsBucket := tx.Bucket([]byte(PacksBucketName))

		if packsBucket.Get([]byte(packName)) != nil {
			return ErrDuplicatePack
		}

		curEpoch := string(metaDataBucket.Get([]byte("epoch")))

		stateBytes, err := json.Marshal(packState{
			UploadComplete: false,
			Epoch:          curEpoch,
		})
		if err != nil {
			return err
		}

		err = packsBucket.Put([]byte(packName), stateBytes)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (d *Drive) FinishUpload(packName string, createdAt time.Time, size uint64) error {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		packsBucket := tx.Bucket([]byte(PacksBucketName))

		oldStateBytes := packsBucket.Get([]byte(packName))
		if oldStateBytes == nil {
			return ErrGCOccurred
		}

		var state packState

		err := json.Unmarshal(oldStateBytes, &state)
		if err != nil {
			return err
		}

		curEpoch := string(metaDataBucket.Get([]byte("epoch")))

		if curEpoch != state.Epoch {
			return ErrGCOccurred
		}

		state.UploadComplete = true
		state.Listing.Date = createdAt
		state.Listing.Size = size

		newStateBytes, err := json.Marshal(state)
		if err != nil {
			return err
		}

		err = packsBucket.Put([]byte(packName), newStateBytes)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (d *Drive) RemovePack(packName, epoch string) error {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))

		if string(metaDataBucket.Get([]byte("gcrunning"))) != "1" {
			return ErrGCNotRunning
		}

		curEpoch := string(metaDataBucket.Get([]byte("epoch")))

		if epoch != curEpoch {
			return ErrGCOccurred
		}

		packsBucket := tx.Bucket([]byte(PacksBucketName))
		err = packsBucket.Delete([]byte(packName))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (d *Drive) GetCompletePacks() ([]PackListing, error) {
	return d.getPacks(false)
}

func (d *Drive) GetAllPacks() ([]PackListing, error) {
	return d.getPacks(true)
}

func (d *Drive) getPacks(allowPartial bool) ([]PackListing, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	listing := make([]PackListing, 0, 32)
	err = db.View(func(tx *bolt.Tx) error {
		packsBucket := tx.Bucket([]byte(PacksBucketName))
		err = packsBucket.ForEach(func(k, v []byte) error {
			var state packState
			err := json.Unmarshal(v, &state)
			if err != nil {
				return err
			}
			if state.UploadComplete || allowPartial {
				state.Listing.Name = string(k)
				listing = append(listing, state.Listing)
			}
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return listing, nil
}
