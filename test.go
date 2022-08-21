package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/go-resty/resty/v2"
)

func main() {
	res, err := resty.New().R().Get("https://www.google.com/ahmadabaffvaf")
	spew.Dump(err)
	spew.Dump(res)

}
