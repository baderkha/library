package main

import (
	"bytes"
	"html/template"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	var b []byte
	b = nil

	s := string(b)
	_ = s
	spew.Dump(s)

	t := template.New("some")
	t, err := t.Parse(`<a href={{.ahmad}}> {{.ahmad}} </a>`)
	spew.Dump(err)
	var buf bytes.Buffer
	t.Execute(&buf, map[string]interface{}{"ahmad": "123"})
	spew.Dump(buf.String())

}
