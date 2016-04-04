package proto9

import (
	"fmt"
	"reflect"
	"testing"
)

type MsgTestCase struct {
	msg  Msg
	data []byte
}

var msgtests = []MsgTestCase{
	{
		msg: &Tversion{
			Tag:         45,
			MessageSize: 9384,
			Version:     "9P2000",
		},
		data: []byte{0x13, 0x0, 0x0, 0x0, 0x64, 0x2d, 0x0, 0xa8, 0x24, 0x0, 0x0, 0x6, 0x0, 0x39, 0x50, 0x32, 0x30, 0x30, 0x30},
	},
	{
		msg: &Tversion{
			Tag:         65535,
			MessageSize: 8192,
			Version:     "9P2000",
		},
		data: []byte{0x13, 0x00, 0x00, 0x00, 0x64, 0xff, 0xff, 0x00, 0x20, 0x00, 0x00, 0x06, 0x00, 0x39, 0x50, 0x32, 0x30, 0x30, 0x30},
	},
	{
		msg: &Rversion{
			Tag:         45,
			MessageSize: 9384,
			Version:     "9P2000",
		},
		data: []byte{0x13, 0x0, 0x0, 0x0, 0x65, 0x2d, 0x0, 0xa8, 0x24, 0x0, 0x0, 0x6, 0x0, 0x39, 0x50, 0x32, 0x30, 0x30, 0x30},
	},

	{
		msg: &Tauth{
			Tag:   45,
			Afid:  1234,
			Uname: "someone",
			Aname: "something",
		},
		data: []byte{0x1f, 0x0, 0x0, 0x0, 0x66, 0x2d, 0x0, 0xd2, 0x4, 0x0, 0x0, 0x7, 0x0, 0x73, 0x6f, 0x6d, 0x65, 0x6f, 0x6e, 0x65, 0x9, 0x0, 0x73, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x69, 0x6e, 0x67},
	},
	/*	{
			msg: &Rauth{
				tag:  45,
				aqid: {ty: 0, vers: 0, path: 0},
			},
			data: {0x14, 0x0, 0x0, 0x0, 0x67, 0x2d, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		{
			msg: &Rauth{
				tag:  0x0101,
				aqid: {ty: 0xff, vers: 0xffffffff, path: 0xffffffffffffffff},
			},
			data: {0x14, 0x0, 0x0, 0x0, 0x67, 0x01, 0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
	*/
	{
		msg: &Rerror{
			Tag: 45,
			Err: "something something something",
		},
		data: []byte{0x26, 0x0, 0x0, 0x0, 0x6b, 0x2d, 0x0, 0x1d, 0x0, 0x73, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x69, 0x6e, 0x67, 0x20, 0x73, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x69, 0x6e, 0x67, 0x20, 0x73, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x69, 0x6e, 0x67},
	},
	/*
		{
			msg: &Tattach{
				tag:   45,
				fid:   35243,
				afid:  90872354,
				uname: "",
				aname: "weee",
			},
			data: []byte{0x17, 0x0, 0x0, 0x0, 0x68, 0x2d, 0x0, 0xab, 0x89, 0x0, 0x0, 0x22, 0x9a, 0x6a, 0x5, 0x0, 0x0, 0x4, 0x0, 0x77, 0x65, 0x65, 0x65},
		},
		{
			msg: &Rattach{
				tag: 45,
				qid: {ty: 0, vers: 0, path: 0},
			},
			data: []byte{0x14, 0x0, 0x0, 0x0, 0x69, 0x2d, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	*/
	{
		msg: &Tflush{
			Tag:    45,
			OldTag: 23453,
		},
		data: []byte{0x9, 0x0, 0x0, 0x0, 0x6c, 0x2d, 0x0, 0x9d, 0x5b},
	},
	{
		msg: &Rflush{
			Tag: 45,
		},
		data: []byte{0x7, 0x0, 0x0, 0x0, 0x6d, 0x2d, 0x0},
	},
	/*
		{
			msg: &Twalk{
				tag:     45,
				fid:     1234,
				newfid:  3452345,
				nwnames: 4,
				wnames: {
					"ongo",
					"bongo",
					"filliyonko",
					"megatronko",
				},
			},
			data: []byte{0x36, 0x0, 0x0, 0x0, 0x6e, 0x2d, 0x0, 0xd2, 0x4, 0x0, 0x0, 0xb9, 0xad, 0x34, 0x0, 0x4, 0x0, 0x4, 0x0, 0x6f, 0x6e, 0x67, 0x6f, 0x5, 0x0, 0x62, 0x6f, 0x6e, 0x67, 0x6f, 0xa, 0x0, 0x66, 0x69, 0x6c, 0x6c, 0x69, 0x79, 0x6f, 0x6e, 0x6b, 0x6f, 0xa, 0x0, 0x6d, 0x65, 0x67, 0x61, 0x74, 0x72, 0x6f, 0x6e, 0x6b, 0x6f},
		},
		{
			msg: &Rwalk{
				tag:    45,
				nnwqid: 1,
				nwqid: {
					{ty: 0, vers: 0, path: 0},
				},
			},
			data: []byte{0x16, 0x0, 0x0, 0x0, 0x6f, 0x2d, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		{
			msg: &Topen{
				tag:  45,
				fid:  21343,
				mode: 4,
			},
			data: []byte{0xc, 0x0, 0x0, 0x0, 0x70, 0x2d, 0x0, 0x5f, 0x53, 0x0, 0x0, 0x4},
		},
		{
			msg: &Ropen{
				tag:    45,
				qid:    {ty: 0, vers: 0, path: 0},
				iounit: 1234123,
			},
			data: []byte{0x18, 0x0, 0x0, 0x0, 0x71, 0x2d, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcb, 0xd4, 0x12, 0x0},
		},
		{
			msg: &Tcreate{
				tag:  45,
				fid:  12343,
				name: "wakakaaka",
				perm: proto9.Dmdir,
				mode: 4,
			},
			data: []byte{0x1b, 0x0, 0x0, 0x0, 0x72, 0x2d, 0x0, 0x37, 0x30, 0x0, 0x0, 0x9, 0x0, 0x77, 0x61, 0x6b, 0x61, 0x6b, 0x61, 0x61, 0x6b, 0x61, 0x0, 0x0, 0x0, 0x80, 0x4},
		},
		{
			msg: &Rcreate{
				tag:    45,
				qid:    {ty: 0, vers: 0, path: 0},
				iounit: 1234123,
			},
			data: []byte{0x18, 0x0, 0x0, 0x0, 0x73, 0x2d, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcb, 0xd4, 0x12, 0x0},
		},
	*/
	{
		msg: &Tread{
			Tag:    45,
			Fid:    5343,
			Offset: 359842382234,
			Count:  23423,
		},
		data: []byte{0x17, 0x0, 0x0, 0x0, 0x74, 0x2d, 0x0, 0xdf, 0x14, 0x0, 0x0, 0x9a, 0x1, 0x47, 0xc8, 0x53, 0x0, 0x0, 0x0, 0x7f, 0x5b, 0x0, 0x0},
	},

	{
		msg: &Rread{
			Tag:  45,
			Data: []byte("ooooh nooo it's full of data"),
		},
		data: []byte{0x27, 0x0, 0x0, 0x0, 0x75, 0x2d, 0x0, 0x1c, 0x0, 0x0, 0x0, 0x6f, 0x6f, 0x6f, 0x6f, 0x68, 0x20, 0x6e, 0x6f, 0x6f, 0x6f, 0x20, 0x69, 0x74, 0x27, 0x73, 0x20, 0x66, 0x75, 0x6c, 0x6c, 0x20, 0x6f, 0x66, 0x20, 0x64, 0x61, 0x74, 0x61},
	},
	/*
		{
			msg: &Twrite{
				tag:  45,
				fid:  254334,
				off:  21304978234,
				data: "something to write",
			},
			data: {0x29, 0x0, 0x0, 0x0, 0x76, 0x2d, 0x0, 0x7e, 0xe1, 0x3, 0x0, 0x3a, 0x2b, 0xe0, 0xf5, 0x4, 0x0, 0x0, 0x0, 0x12, 0x0, 0x0, 0x0, 0x73, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x69, 0x6e, 0x67, 0x20, 0x74, 0x6f, 0x20, 0x77, 0x72, 0x69, 0x74, 0x65},
		},
		{
			msg: &Rwrite{
				tag:   45,
				count: 12,
			},
			data: {0xb, 0x0, 0x0, 0x0, 0x77, 0x2d, 0x0, 0xc, 0x0, 0x0, 0x0},
		},
		{
			msg: &Tclunk{
				tag: 45,
				fid: 23123,
			},
			data: {0xb, 0x0, 0x0, 0x0, 0x78, 0x2d, 0x0, 0x53, 0x5a, 0x0, 0x0},
		},
		{
			msg: &Rclunk{
				tag: 45,
			},
			data: {0x7, 0x0, 0x0, 0x0, 0x79, 0x2d, 0x0},
		},
		{
			msg: &Tremove{
				tag: 45,
				fid: 1234,
			},
			data: {0xb, 0x0, 0x0, 0x0, 0x7a, 0x2d, 0x0, 0xd2, 0x4, 0x0, 0x0},
		},
		{
			msg: &Rremove{
				tag: 45,
			},
			data: {0x7, 0x0, 0x0, 0x0, 0x7b, 0x2d, 0x0},
		},
		{
			msg: &Tstat{
				tag: 45,
				fid: 12341234,
			},
			data: {0xb, 0x0, 0x0, 0x0, 0x7c, 0x2d, 0x0, 0xf2, 0x4f, 0xbc, 0x0},
		},
		{
			msg: &Rstat{
				tag: 45,
				stat: {
					ty:    0,
					dev:   0,
					qid:   {ty: 0, vers: 0, path: 0},
					mode:  0,
					atime: 0,
					mtime: 0,
					len:   0,
					name:  "",
					uid:   "",
					gid:   "",
					muid:  "",
				},
			},
			data: {0x3a, 0x0, 0x0, 0x0, 0x7d, 0x2d, 0x0, 0x31, 0x0, 0x2f, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		{
			msg: &Rstat{
				tag: 45,
				stat: {
					ty:    0xffff,
					dev:   0xffffffff,
					qid:   {ty: 0xff, vers: 0xffffffff, path: 0xffffffffffffffff},
					mode:  0xffffffff,
					atime: 0xffffffff,
					mtime: 0xffffffff,
					len:   0xffffffffffffffff,
					name:  "x",
					uid:   "x",
					gid:   "x",
					muid:  "x",
				},
			},
			data: {0x3e, 0x0, 0x0, 0x0, 0x7d, 0x2d, 0x0, 0x35, 0x0, 0x33, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0x1, 0x0, 0x78, 0x1, 0x0, 0x78, 0x1, 0x0, 0x78, 0x1, 0x0, 0x78},
		},
		{
			msg: &Twstat{
				tag: 45,
				fid: 12342134,
				stat: {
					ty:    0,
					dev:   0,
					qid:   {ty: 0, vers: 0, path: 0},
					mode:  0,
					atime: 0,
					mtime: 0,
					len:   0,
					name:  "",
					uid:   "",
					gid:   "",
					muid:  "",
				},
			},
			data: {0x3e, 0x0, 0x0, 0x0, 0x7e, 0x2d, 0x0, 0x76, 0x53, 0xbc, 0x0, 0x31, 0x0, 0x2f, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		{
			msg: &Rwstat{
				tag: 45,
			},
			data: {0x7, 0x0, 0x0, 0x0, 0x7f, 0x2d, 0x0},
		},
	*/
}

func TestPackUnpack(t *testing.T) {
	for _, tc := range msgtests {
		l := tc.msg.WireLen()
		if l != len(tc.data) {
			t.Fatal("WireLen incorrect")
		}
		buf := make([]byte, l, l)
		PackMsg(buf, tc.msg)
		if !reflect.DeepEqual(buf, tc.data) {
			fmt.Printf("msg=%#v\nexp=%v\ngot=%v\n", tc.msg, tc.data, buf)
			t.Fatal("Pack incorrect")
		}
		unpacked, err := UnpackMsg(tc.data)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tc.msg, unpacked) {
			fmt.Printf("exp=%v\ngot=%v\n", tc.msg, unpacked)
			t.Fatal("Unpack incorrect")
		}
	}
}
