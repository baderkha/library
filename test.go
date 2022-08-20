package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

func main() {
	s := "Hello"

	key := "FDJ1mnhuzjFjTdwhq7DtZG2Cq9kuuEZCG"
	h := hmac.New(sha256.New, []byte(key))

	h.Write([]byte(s))
	fmt.Printf("%x\n", h.Sum(nil))

	fmt.Printf("%x\n", sha256.Sum256([]byte(s)))
}
