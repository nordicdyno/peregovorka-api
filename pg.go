package main

import (
	"encoding/json"
	"log"
)

type UserDbInfo struct {
	Status       int
	Name         string
	Nick         string
	RateStatusId int `db:"rate_status_id"`
	//Media string
	Media MediaData
}

type MediaData struct {
	AvatarLink string
}

type JsonMedia map[string]ImagesMap

type ImagesMap map[string]ImageData

type ImageData struct {
	Width    string
	Height   string
	Filelink string
}

// http://go-database-sql.org/
// http://jmoiron.net/blog/gos-database-sql/
// http://webgo.io/api.html
// http://jmoiron.net/blog/built-in-interfaces/
// http://stackoverflow.com/questions/25758138/storing-golang-json-into-postgresql
// http://nathanleclaire.com/blog/2013/11/04/want-to-work-with-databases-in-golang-lets-try-some-gorp/
func (g *MediaData) Scan(src interface{}) error {
	g.AvatarLink = "http://s5o.ru/common/images/blank_icons/user_small.png"

	var jsonRaw []byte
	switch src.(type) {
	case []byte:
		//log.Println("byte")
		jsonRaw = src.([]byte)
	default:
		log.Println("not byte[] for Media")
		return nil
	}

	j := make(JsonMedia)
	json.Unmarshal(jsonRaw, &j)
	// log.Println(spew.Sdump(jsonRaw))
	// log.Println(spew.Sdump(j))

	imgs, ok := j["avatar"]
	if !ok {
		return nil
	}

	data, ok := imgs["webdav"]
	if !ok {
		return nil
	}

	g.AvatarLink = data.Filelink
	return nil
}
