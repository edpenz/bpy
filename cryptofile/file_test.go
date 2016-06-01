package cryptofile

import (
	"bytes"
	"io"
	"math/rand"
	"reflect"
	"testing"
)

type XorBlock struct {
	BlockSz int
}

func (xb *XorBlock) BlockSize() int { return xb.BlockSz }

func (xb *XorBlock) Encrypt(dst, src []byte) {
	for i := range src {
		dst[i] = src[i] ^ 0xf0
	}
}

func (xb *XorBlock) Decrypt(dst, src []byte) {
	xb.Encrypt(dst, src)
}

func TestReadWrite(t *testing.T) {

	random := rand.New(rand.NewSource(4532))

	for _, blocksz := range []int{1, 2, 8, 32} {
		for sz := 0; sz < 1000; sz++ {
			var buf bytes.Buffer

			data := make([]byte, sz, sz)
			_, err := io.ReadFull(random, data)
			if err != nil {
				t.Fatal(err)
			}

			block := &XorBlock{BlockSz: blocksz}
			w := NewWriter(&buf, block, make([]byte, blocksz, blocksz))

			ncopied := 0
			for ncopied != len(data) {
				amnt := rand.Int() % (blocksz * 3)
				if ncopied+amnt > len(data) {
					amnt = len(data) - ncopied
				}
				n, err := w.Write(data[ncopied : ncopied+amnt])
				if err != nil {
					t.Fatal(err)
				}
				ncopied += n
			}

			err = w.Close()
			if err != nil {
				t.Fatal(err)
			}

			if buf.Len()%blocksz != 0 {
				t.Fatal("len is not a multiple of block size")
			}

			rdr := NewReader(bytes.NewReader(buf.Bytes()), block, make([]byte, blocksz, blocksz), int64(buf.Len()))
			result := make([]byte, len(data), len(data))
			nread := 0
			for nread != len(data) {
				amnt := rand.Int() % (blocksz * 3)
				if nread+amnt > len(data) {
					amnt = len(data) - nread
				}
				n, err := io.ReadFull(rdr, result[nread:nread+amnt])
				if err != nil {
					t.Fatal(err)
				}
				nread += n
			}

			n, err := io.ReadFull(rdr, make([]byte, 10000, 10000))
			if n != 0 || err != io.EOF {
				t.Fatal("expected EOF")
			}

			if !reflect.DeepEqual(data, result) {
				t.Fatalf("data differs: %v != %v", result, data)
			}
		}
	}

}