package sig

import (
	"github.com/buppyio/bpy"
	"reflect"
	"testing"
)

func TestSigs(t *testing.T) {
	k, err := bpy.NewKey()
	if err != nil {
		t.Fatal(err)
	}

	h := [32]byte{}
	signed := SignHash(&k, 123, h)
	version, got, err := ParseSignedHash(&k, signed)
	if err != nil {
		t.Fatal(err)
	}

	if version != 123 {
		t.Fatal("Bad version")
	}

	if !reflect.DeepEqual(h, got) {
		t.Fatal("parsing ref failed")
	}

	for i := 0; i < len(signed); i++ {
		corrupt := []byte(signed)
		corrupt[i] = corrupt[i] + 1
		_, _, err := ParseSignedHash(&k, string(corrupt))
		if err == nil {
			t.Fatal("expected failure")
		}
	}
}
