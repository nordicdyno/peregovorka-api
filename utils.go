package main

import (
	"crypto/sha1"
	"fmt"
	"io"

	//"github.com/davecgh/go-spew/spew"
	//"log"
)

func makeRoomId(owner string, guests []string, size int) string {
	h := sha1.New()
	io.WriteString(h, owner)
	for _, gst := range guests {
		io.WriteString(h, gst)
	}
	s := fmt.Sprintf("%x", h.Sum(nil))
	if size > 0 && len(s) > size {
		s = s[:size]
	}
	//log.Println(owner, "+", spew.Sdump(guests), "=>", s)
	return s
}
