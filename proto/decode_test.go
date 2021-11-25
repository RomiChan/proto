package proto

import (
	"errors"
	"io"
	"testing"
)

func TestUnarshalFromShortBuffer(t *testing.T) {
	m := message{
		A: 1,
		B: 2,
		C: 3,
		S: submessage{
			X: "hello",
			Y: "world",
		},
	}

	b, err := Marshal(&m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	for i := range b {
		switch i {
		case 0, 2, 4, 6:
			continue // these land on field boundaries, making the input valid
		}
		t.Run("", func(t *testing.T) {
			msg := &message{}
			err := Unmarshal(b[:i], msg)
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Errorf("error mismatch, want io.ErrUnexpectedEOF but got %q", err)
			}
		})
	}
}

func BenchmarkDecodeTag(b *testing.B) {
	c := appendTag(nil, 1, varint)

	for i := 0; i < b.N; i++ {
		decodeTag(c)
	}
}

func BenchmarkDecodeMessage(b *testing.B) {
	data, _ := Marshal(message{
		A: 1,
		B: 100,
		C: 10000,
		S: submessage{
			X: "",
			Y: "Hello World!",
		},
	})

	msg := message{}
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		if err := Unmarshal(data, &msg); err != nil {
			b.Fatal(err)
		}
		msg = message{}
	}
}

func BenchmarkDecodeMap(b *testing.B) {
	type message struct {
		M map[int]int
	}

	data, _ := Marshal(message{
		M: map[int]int{
			0: 0,
			1: 1,
			2: 2,
			3: 3,
			4: 4,
		},
	})

	msg := message{}
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		if err := Unmarshal(data, &msg); err != nil {
			b.Fatal(err)
		}
		msg = message{}
	}
}

func BenchmarkDecodeSlice(b *testing.B) {
	type message struct {
		S []int
	}

	data, _ := Marshal(message{
		S: []int{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		},
	})

	msg := message{}
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		if err := Unmarshal(data, &msg); err != nil {
			b.Fatal(err)
		}
		msg = message{}
	}

}

func TestIssue110(t *testing.T) {
	type message struct {
		A *uint32 `protobuf:"fixed32,1,opt"`
	}

	var a uint32 = 0x41c06db4
	data, _ := Marshal(message{
		A: &a,
	})

	var m message
	err := Unmarshal(data, &m)
	if err != nil {
		t.Fatal(err)
	}
	if *m.A != 0x41c06db4 {
		t.Errorf("m.A mismatch, want 0x41c06db4 but got %x", m.A)
	}
}
