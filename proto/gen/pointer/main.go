package main

import (
	"bytes"
	"go/format"
	"os"
	"strings"
	"text/template"
)

const decodeTmpl = `// Code generated by gen/gen_pointer.go. DO NOT EDIT.

package proto

import "unsafe"

{{range .Decoder}}
var {{.Codec}}PtrCodec = codec{
	size:   sizeOf{{.Name}}Ptr,
	encode: encode{{.Name}}Ptr,
	decode: decode{{.Name}}Ptr,
}

func sizeOf{{.Name}}Ptr(p unsafe.Pointer, f *structField) int {
	p = deref(p)
	if p != nil {
		return sizeOf{{.Name}}Required(p, f)
	}
	return 0
}

func encode{{.Name}}Ptr(b []byte, p unsafe.Pointer, f *structField) []byte {
	p = deref(p)
	if p != nil {
		return encode{{.Name}}Required(b, p, f)
	}
	return b
}

func decode{{.Name}}Ptr(b []byte, p unsafe.Pointer) (int, error) {
	v := (*unsafe.Pointer)(p)
	if *v == nil {
		*v = unsafe.Pointer(new({{.Type}}))
	}
	return decode{{.Name}}(b, *v)
}

{{end}}
`

func main() {
	type decoder struct {
		Type  string
		Name  string
		Codec string
	}

	var decoders = []decoder{
		{Type: "bool", Name: "Bool"},
		{Type: "string", Name: "String"},
		{Type: "int32", Name: "Int32"},
		{Type: "uint32", Name: "Uint32"},
		{Type: "int64", Name: "Int64"},
		{Type: "uint64", Name: "Uint64"},
		{Type: "int32", Name: "Zigzag32"},
		{Type: "int64", Name: "Zigzag64"},
		{Type: "uint32", Name: "Fixed32"},
		{Type: "uint64", Name: "Fixed64"},
		{Type: "float32", Name: "Float32"},
		{Type: "float64", Name: "Float64"},
	}

	for i, d := range decoders {
		decoders[i].Codec = strings.ToLower(d.Name)
	}

	var out bytes.Buffer
	tmpl, err := template.New("").Parse(decodeTmpl)
	if err != nil {
		panic(err)
	}
	tmpl.Execute(&out, &struct {
		Decoder []decoder
	}{
		Decoder: decoders,
	})

	source, err := format.Source(out.Bytes())
	if err != nil {
		panic(err)
	}
	f, _ := os.OpenFile("pointer_codec.go", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_TRUNC, 0o644)
	f.Write(source)
}
