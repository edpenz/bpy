package drive

import (
	"github.com/buppyio/bpy"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAttach(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	drive, err := Open(filepath.Join(testDir, "drive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer drive.Close()

	ok, err := drive.Attach("abc")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected ok attach")
	}

	ok, err = drive.Attach("abd")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected !ok attach")
	}

	ok, err = drive.Attach("abc")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected ok attach")
	}
}

func TestGCGeneration(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	drive, err := Open(filepath.Join(testDir, "drive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer drive.Close()

	gen1, err := drive.GetGCGeneration()
	if err != nil {
		t.Fatal(err)
	}

	gen2, err := drive.StartGC()
	if err != nil {
		t.Fatal(err)
	}
	if gen2 == gen1 {
		t.Fatal("gcGeneration did not increment")
	}

	err = drive.StopGC()
	if err != nil {
		t.Fatal(err)
	}

	gen3, err := drive.GetGCGeneration()
	if err != nil {
		t.Fatal(err)
	}
	if gen3 == gen2 {
		t.Fatal("unexpected gcGeneration")
	}
}

func TestCasRoot(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	drive, err := Open(filepath.Join(testDir, "drive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer drive.Close()

	gcGeneration, err := drive.GetGCGeneration()
	if err != nil {
		t.Fatal(err)
	}

	root, version, sig, err := drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "" || sig != "" {
		t.Fatal("unexpected root/sig")
	}

	ok, err := drive.CasRoot("foo", bpy.NextRootVersion(version), "sig", "bad gen")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	ok, err = drive.CasRoot("foo", "", "sig", gcGeneration)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	root, version, sig, err = drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "" || sig != "" {
		t.Fatal("unexpected root/sig")
	}

	ok, err = drive.CasRoot("foo", bpy.NextRootVersion(version), "sig", gcGeneration)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("unexpected !ok")
	}

	root, version, sig, err = drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "foo" || sig != "sig" {
		t.Fatalf("unexpected root/sig=%s/%d/%s", root, sig)
	}
}

func TestUploadPack(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	drive, err := Open(filepath.Join(testDir, "drive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer drive.Close()

	err = drive.StartUpload("foobar")
	if err != nil {
		t.Fatal(err)
	}
	err = drive.FinishUpload("foobar", time.Now(), 0)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.StartUpload("foobar")
	if err != ErrDuplicatePack {
		t.Fatal("expected duplicate pack error, got:", err)
	}

	packs, err := drive.GetPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 || packs[0].Name != "foobar" {
		t.Fatal("expected one pack called foobar")
	}

}

func TestRemovePack(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	drive, err := Open(filepath.Join(testDir, "drive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer drive.Close()

	gcGeneration, err := drive.GetGCGeneration()
	if err != nil {
		t.Fatal(err)
	}

	err = drive.StartUpload("foobar")
	if err != nil {
		t.Fatal(err)
	}
	err = drive.FinishUpload("foobar", time.Now(), 0)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.StartUpload("foobarbaz")
	if err != nil {
		t.Fatal(err)
	}
	err = drive.FinishUpload("foobarbaz", time.Now(), 0)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.RemovePack("foobar", gcGeneration)
	if err != ErrGCNotRunning {
		t.Fatal("expected ErrGCNotRunning error, got: ", err)
	}

	packs, err := drive.GetPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 2 {
		t.Fatal("expected remove failed")
	}

	gcGeneration, err = drive.StartGC()
	if err != nil {
		t.Fatal(err)
	}

	err = drive.RemovePack("foobar", gcGeneration)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.StopGC()
	if err != nil {
		t.Fatal(err)
	}

	packs, err = drive.GetPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 || packs[0].Name != "foobarbaz" {
		t.Fatal("expected remove success")
	}

}
