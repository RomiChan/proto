package proto

import (
	"reflect"
	"sync"
	"unsafe"

	. "github.com/RomiChan/protobuf/internal/runtime_reflect"
)

const (
	zeroSize = 1 // sizeOfVarint(0)
)

type mapField struct {
	number   uint16
	keyFlags uint8
	valFlags uint8
	keyCodec *codec
	valCodec *codec
}

func mapCodecOf(t reflect.Type, f *mapField, seen map[reflect.Type]*codec) *codec {
	m := new(codec)
	seen[t] = m

	m.wire = varlen
	m.size = mapSizeFuncOf(t, f)
	m.encode = mapEncodeFuncOf(t, f)
	m.decode = mapDecodeFuncOf(t, f, seen)
	return m
}

func mapSizeFuncOf(t reflect.Type, f *mapField) sizeFunc {
	mapTagSize := sizeOfTag(fieldNumber(f.number), varlen)
	keyTagSize := sizeOfTag(1, f.keyCodec.wire)
	valTagSize := sizeOfTag(2, f.valCodec.wire)
	return func(p unsafe.Pointer, flags flags) int {
		if p == nil {
			return 0
		}

		if !flags.has(inline) {
			p = *(*unsafe.Pointer)(p)
		}

		n := 0
		m := MapIter{}
		defer m.Done()

		for m.Init(pointer(t), p); m.HasNext(); m.Next() {
			keySize := f.keyCodec.size(m.Key(), wantzero)
			valSize := f.valCodec.size(m.Value(), wantzero)

			if keySize > 0 {
				n += keyTagSize + keySize
				if (f.keyFlags & embedded) != 0 {
					n += sizeOfVarint(uint64(keySize))
				}
			}

			if valSize > 0 {
				n += valTagSize + valSize
				if (f.valFlags & embedded) != 0 {
					n += sizeOfVarint(uint64(valSize))
				}
			}

			n += mapTagSize + sizeOfVarint(uint64(keySize+valSize))
		}

		if n == 0 {
			n = mapTagSize + zeroSize
		}

		return n
	}
}

func mapEncodeFuncOf(t reflect.Type, f *mapField) encodeFunc {
	keyTag := byte(uint64(1)<<3 | uint64(f.keyCodec.wire))
	valTag := byte(uint64(2)<<3 | uint64(f.valCodec.wire))

	mapTag := appendTag(nil, fieldNumber(f.number), varlen)
	zero := append(mapTag, 0)

	return func(b []byte, p unsafe.Pointer, flags flags) ([]byte, error) {
		if p == nil {
			return b, nil
		}

		if !flags.has(inline) {
			p = *(*unsafe.Pointer)(p)
		}

		origLen := len(b)
		var err error

		m := MapIter{}
		defer m.Done()

		for m.Init(pointer(t), p); m.HasNext(); m.Next() {
			key := m.Key()
			val := m.Value()

			keySize := f.keyCodec.size(key, wantzero)
			valSize := f.valCodec.size(val, wantzero)
			elemSize := keySize + valSize

			if keySize > 0 {
				elemSize += 1 // keyTagSize
				if (f.keyFlags & embedded) != 0 {
					elemSize += sizeOfVarint(uint64(keySize))
				}
			}

			if valSize > 0 {
				elemSize += 1 // valTagSize
				if (f.valFlags & embedded) != 0 {
					elemSize += sizeOfVarint(uint64(valSize))
				}
			}

			b = append(b, mapTag...)
			b = appendVarint(b, uint64(elemSize))

			if keySize > 0 {
				b = append(b, keyTag)

				if (f.keyFlags & embedded) != 0 {
					b = appendVarint(b, uint64(keySize))
				}

				b, err = f.keyCodec.encode(b, key, wantzero)
				if err != nil {
					return b, err
				}
			}

			if valSize > 0 {
				b = append(b, valTag)

				if (f.valFlags & embedded) != 0 {
					b = appendVarint(b, uint64(valSize))
				}

				b, err = f.valCodec.encode(b, val, wantzero)
				if err != nil {
					return b, err
				}
			}
		}

		if len(b) == origLen {
			b = append(b, zero...)
		}
		return b, nil
	}
}

func mapDecodeFuncOf(t reflect.Type, _ *mapField, seen map[reflect.Type]*codec) decodeFunc {
	structType := reflect.StructOf([]reflect.StructField{
		{Name: "Key", Type: t.Key()},
		{Name: "Elem", Type: t.Elem()},
	})

	structCodec := codecOf(structType, seen)
	structPool := new(sync.Pool)
	structZero := pointer(reflect.Zero(structType).Interface())

	valueType := t.Elem()
	valueOffset := structType.Field(1).Offset

	mtype := pointer(t)
	stype := pointer(structType)
	vtype := pointer(valueType)

	return func(b []byte, p unsafe.Pointer, _ flags) (int, error) {
		m := (*unsafe.Pointer)(p)
		if *m == nil {
			*m = MakeMap(mtype, 10)
		}
		if len(b) == 0 {
			return 0, nil
		}

		s := pointer(structPool.Get())
		if s == nil {
			s = unsafe.Pointer(reflect.New(structType).Pointer())
		}

		n, err := structCodec.decode(b, s, noflags)
		if err == nil {
			v := MapAssign(mtype, *m, s)
			Assign(vtype, v, unsafe.Pointer(uintptr(s)+valueOffset))
		}

		Assign(stype, s, structZero)
		structPool.Put(s)
		return n, err
	}
}
