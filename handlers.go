package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrNotFound = errors.New("handlers: data not found")
)

func listHistoryRoomsHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("call listHistoryRoomsHandler")
	roomid := req.PostFormValue("roomid")
	uid, err := checkAndGetUser(req)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.NotFound(w, req)
			// io.WriteString(w, "{}") // 404 ?
			return
		}
		panic(err)
	}

	history := RoomData{
		// init Guests & Messages
	}
	selector := bson.M{"id": roomid, "ownerid": uid}
	var projection interface {}
	projection = bson.M{"messages": bson.M{"$slice": -20}}
	log.Println("selector: ", selector)
	err = queryMongoCollectionOne("rooms", selector, projection, &history)
	if err != nil {
		if err == mgo.ErrNotFound {
			log.Println("Not found history for room =", roomid,"& uid =", uid)
			w.Header().Set("Content-Type", "application/json;charset=utf-8")
			io.WriteString(w, "{}") // 404 ?
			return
		}
		panic(err.Error() + ": " + string(debug.Stack()))
	}
	log.Println("room history: ", spew.Sdump(&history))

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	jEnc := json.NewEncoder(w)
	jEnc.Encode(&history)
}

func sendRoomHandler(w http.ResponseWriter, req *http.Request) {
	var (
		uid string
		err error
	)
	log.Println("call sendRoomHandler")
	uid, err = checkAndGetUser(req)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.NotFound(w, req)
			return
		}
		panic(err)
	}

	roomid := req.PostFormValue("roomid")
	content := req.PostFormValue("content")

	message := Message{
		Dt:       time.Now(),
		Content:  content,
		AuthorId: uid,
		Readed:   false,
	}

	err = mongoSendMessage(roomid, &message)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(w, "{}")
}

func listRoomsHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("call listRoomsHandler")
	uid, err := checkAndGetUser(req)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.NotFound(w, req)
			return
		}
		panic(err)
	}

	house := HouseData{
		Rooms: make(map[string]*RoomDataDesc),
	}
	selector := bson.M{"uid": uid}
	log.Println("selector: ", selector)
	//err := queryMongoCollectionOne("house", selector, &house)
	err = queryMongoCollectionOne("houses", selector, nil, &house)
	if err != nil {
		if err == mgo.ErrNotFound {
			log.Println("Not found house for uid=", uid)
			w.Header().Set("Content-Type", "application/json;charset=utf-8")
			io.WriteString(w, "{}") // 404 ?
			return
		}
		panic(err.Error() + ": " + string(debug.Stack()))
	}
	log.Println("go house: ", spew.Sdump(&house))

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	jEnc := json.NewEncoder(w)
	jEnc.Encode(&house)
}

// createRoomHandler create room for current user id to specified user id (guidPOST param)
func createRoomHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("call createRoomHandler")
	uid, err := checkAndGetUser(req)
	if err != nil {
		if err == mgo.ErrNotFound {
			http.NotFound(w, req)
			// io.WriteString(w, "{}") // 404 ?
			return
		}
		panic(err)
	}

	guid := req.PostFormValue("guid")
	if uid == guid {
		panic("Ids can't match")
	}
	// FIXME: check if guid exists

	guestIds := []string{guid}

	roomid, err := mongoCreateRoom(uid, guestIds)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	io.WriteString(w, "{\"roomid\": \""+roomid+"\"}")
}

// registerUserHandler user register or update data in chat
func registerUserHandler(w http.ResponseWriter, req *http.Request) {
	id_str, err := checkAndGetUser(req)
	if err != nil {
		if err == ErrNotFound {
			http.NotFound(w, req)
			return
		}
		panic(err)
	}

	id, err := strconv.Atoi(id_str)
	if err != nil {
		panic("can't convert " + id_str + err.Error())
	}

	// TODO: move to function
	log.Println("before query")
	var rows *sqlx.Rows
	rows, err = globals.DbLink.Queryx(
		`SELECT status, name, nick, rate_status_id, media FROM sport_users WHERE id = $1`,
		id,
	)
	// _ = rows
	if err, ok := err.(*pq.Error); ok {
		log.Panic("pq error: ", err.Code.Name(), " >>> ", err.Error())
	}
	defer rows.Close()
	log.Println("after query")

	userDb := UserDbInfo{}
	for rows.Next() {
		err := rows.StructScan(&userDb)
		if err != nil {
			log.Panic(err)
		}
		log.Println(spew.Sdump(userDb))
		break
	}
	err = rows.Err()
	if err != nil {
		log.Panic(err)
	}

	user := UserData{
		Uid:       id_str,
		LastLogin: time.Now(),
		Online:    true,
		Name:      userDb.Name,
		AvatarUrl: userDb.Media.AvatarLink,
	}
	err = mongoAddUser(&user)
	if err != nil {
		panic(err)
	}
}

// listUsersHandler chat users list
func listUsersHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("call listUsersHandler")
	list := make(UserList, 100)
	err := queryMongoCollectionAll("users", nil, &list)
	if err != nil {
		panic(err)
	}
	//log.Println(spew.Sdump(&list))

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	jEnc := json.NewEncoder(w)
	jEnc.Encode(&list)
}

// checkAndGetUser helper function for cookie check in memcache
func checkAndGetUser(req *http.Request) (string, error) {
	sidCookie, err := req.Cookie("sid")
	if err != nil {
		log.Println("auth cookie not set:", err)
		return "", ErrNotFound
	}

	log.Println("sidCookie:", spew.Sdump(sidCookie))
	sid := sidCookie.Value

	log.Println("sid is: ", sid)

	// TODO: move to func
	key := "x|session_" + sid
	it, err := globals.MemClient.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			log.Println("user session not found for id =", sid)
			return "", ErrNotFound
		}
		// check other errors
		return "", err
	}

	var id_str string
	data := string(it.Value)
	for _, val := range strings.Split(data, "\n") {
		pair := strings.Split(val, "\t")
		if pair[0] == "user_id" {
			id_str = pair[1]
			break
		}
	}
	return id_str, nil
}

// Debug handlers

func createUserHandlerDebug(w http.ResponseWriter, req *http.Request) {
	log.Println("call createUserHandler")
	uid := req.PostFormValue("uid")
	name := req.PostFormValue("name")
	avatar_url := req.PostFormValue("avatar_url")

	user := UserData{
		Uid:       uid,
		LastLogin: time.Now(),
		Online:    true,
		Name:      name,
		AvatarUrl: avatar_url,
	}
	err := mongoAddUser(&user)
	if err != nil {
		panic(err)
	}
}

func createRoomHandlerDebug(w http.ResponseWriter, req *http.Request) {
	log.Println("call createRoomHandlerDebug")
	uid := req.PostFormValue("uid")
	guid := req.PostFormValue("guid")
	if uid == guid {
		panic("Ids can't match")
	}

	guestIds := []string{guid}
	var id string
	id = makeRoomId(uid, guestIds, shaLength)

	roomBson := bson.M{
		"id":      id,
		"ownerid": uid,
		"guests":  guestIds,
	}
	selector := bson.M{"id": id}
	log.Println("rooms selector: ", selector)
	log.Println("rooms update: ", roomBson)

	res, err := upsertMongoCollection("rooms", selector, roomBson)
	if err != nil {
		panic(err)
	}
	//_ = res
	log.Println("rooms result:", spew.Sdump(res))

	houseSelector := bson.M{
		"uid": uid,
	}
	roomSubKey := "rooms." + id
	houseBson := bson.M{
		"$set": bson.M{
			roomSubKey: roomBson,
		},
	}

	log.Println("houses selector: ", houseSelector)
	log.Println("houses update: ", houseBson)

	res, err = upsertMongoCollection("houses", houseSelector, houseBson)
	if err != nil {
		panic(err)
	}
	log.Println("houses result:", spew.Sdump(res))
}
