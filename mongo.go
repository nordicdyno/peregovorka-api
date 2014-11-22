package main

import (
	"log"
	"sort"
	"time"

	"github.com/davecgh/go-spew/spew"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	//"strings"
)

type UserList []*UserData

type UserData struct {
	Uid       string    `json:"uid"`
	LastLogin time.Time `bson:"last_login_dt" json:"last_login_dt"`
	Online    bool      `json:"online"`
	Name      string    `json:"name"`
	AvatarUrl string    `bson:"avatar_url" json:"avatar_url"`
}

type HouseData struct {
	Uid   string
	Rooms map[string]*RoomDataDesc
}

type RoomDataDesc struct {
	Id          string
	OwnerId     string
	LastMessage *Message
	LastUser    *UserData
	Unreaded    int32
}

type RoomData struct {
	Id       string
	OwnerId  string
	Guests   map[string]bool
	Messages []*Message
	Unreaded int32
}

type Message struct {
	Id       string
	Dt       time.Time
	Content  string
	AuthorId string
	Readed   bool
}

func mongoSendMessage(roomid string, message *Message) error {
	// TODO: check room existence
	roomSelector := bson.M{"id": roomid}
	log.Println("room selector: ", roomSelector)
	// get Room object here
	baseRoom := RoomData{}
	err := queryMongoCollectionOne("rooms", roomSelector, nil, &baseRoom)
	if err != nil {
		return err
	}

	log.Println("mongoSendMessage: found room", spew.Sdump(baseRoom))

	// allRoomIds: owner Id -> other users []Id map
	allRoomIds := make(map[string][]string)

	baseGuestsIds := make([]string, 0, len(baseRoom.Guests))
	for k := range baseRoom.Guests {
		baseGuestsIds = append(baseGuestsIds, k)
	}
	allRoomIds[baseRoom.OwnerId] = baseGuestsIds

	for guestId := range baseRoom.Guests {
		keys := []string{baseRoom.OwnerId}
		// for multy chats
		for k := range baseRoom.Guests {
			if k == guestId {
				break
			}
			keys = append(keys, k)
		}
		// FIXME: probably move sort to makeRoomId ?
		sort.Strings(keys)
		allRoomIds[guestId] = keys
	}
	log.Println("mongoSendMessage: All rooms", spew.Sdump(allRoomIds))

	for ownerid, guests := range allRoomIds {
		id := makeRoomId(ownerid, guests, shaLength)
		// check if id exists

		err = mongoAddMessageToRoom(ownerid, id, message)
		if err != nil {
			log.Println("Error:", err)
			continue
		}
		err = mongoAddMessageToHouseRoom(ownerid, id, message)
		if err != nil {
			log.Println("Error:", err)
			continue
		}
	}

	// TODO: concurently update all rooms
	// TODO: if any update failed will do anything (requeue, message queues, etc)

	log.Println("mongoSendMessage: update ok")
	return nil
}

func mongoCreateRoom(ownerid string, guests []string) (string, error) {
	// FIXME: temporary hack
	guid := guests[0]
	guestIds := []string{guid}
	log.Println("call mongoCreateRoom()")
	roomid := makeRoomId(ownerid, guestIds, shaLength)
/*
	type RoomDataDesc struct {
		Id          string
		OwnerId     string
		LastMessage *Message
		LastUser    *UserData
		Unreaded    int32
	}
*/
	// FIXME : temporary hack?
	user := UserData{}
	err := queryMongoCollectionOne("users", bson.M{"uid": guid}, nil, &user)
	if err != nil {
		return "", err
	}

	updateMap := make(map[string]interface{})
	// TODO: add loop for multy chats
	updateMap["id"] = roomid
	updateMap["ownerid"] = ownerid
	updateMap["guests."+guid] = true
	roomBson := bson.M{
		"$set": bson.M(updateMap),
	}
	updateMap["lastuser"] = &user

	selector := bson.M{"id": roomid}
	log.Println("rooms selector: ", selector)
	log.Println("rooms update: ", roomBson)

	res, err := upsertMongoCollection("rooms", selector, roomBson)
	if err != nil {
		return "", err
	}
	//_ = res
	log.Println("mongoCreateRoom create room result:", spew.Sdump(res))

	houseSelector := bson.M{"uid": ownerid}
	roomSubKey := "rooms." + roomid
	houseBson := bson.M{
		"$set": bson.M{
			roomSubKey: bson.M{"id": roomid, "lastuser": &user},
		},
	}
	log.Println("mongoCreateRoom houses selector: ", houseSelector)
	log.Println("mongoCreateRoom houses update: ", houseBson)
	res, err = upsertMongoCollection("houses", houseSelector, houseBson)
	if err != nil {
		return "", err
	}
	log.Println("mongoCreateRoom houses result:", spew.Sdump(res))

	return roomid, nil
}

func mongoAddMessageToRoom(uid string, roomid string, message *Message) error {
	roomSelector := bson.M{"id": roomid}
	msgUpdateBson := bson.M{
		"$push": bson.M{
			"messages": bson.M{
				"$each":  []*Message{message},
				"$slice": -500,
			},
		},
	}
	log.Println("add message to room: ", roomid, "data:", spew.Sdump(msgUpdateBson))

	_, err := upsertMongoCollection("rooms", roomSelector, msgUpdateBson)
	if err != nil {
		return err
	}
	return nil
}

func mongoAddMessageToHouseRoom(uid string, roomid string, message *Message) error {
	houseSelector := bson.M{"uid": uid}
	msgUpdateBson := bson.M{
		"$set": bson.M{
			"uid":             uid,
			"rooms." + roomid + ".last_message": message,
		},
	}
	log.Println("add message data to house: ", spew.Sdump(msgUpdateBson))

	_, err := upsertMongoCollection("houses", houseSelector, msgUpdateBson)
	if err != nil {
		return err
	}
	return nil
}

func mongoAddUser(user *UserData) error {
	selector := bson.M{"uid": user.Uid}
	data := bson.M{"$set": user}
	res, err := upsertMongoCollection("users", selector, &data)
	if err != nil {
		panic(err)
	}

	log.Println("result:", spew.Sdump(res))
	return nil
}

func queryMongoCollectionAll(name string, query interface{}, result interface{}) error {
	session, err := mgo.DialWithInfo(globals.MongoDialInfo)
	if err != nil {
		return err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	collection := session.DB(conf.Mongo.Database).C(name)
	log.Println("before find")
	return collection.Find(query).All(result)
}

func queryMongoCollectionOne(name string, criteria interface{}, projection interface{}, result interface{}) error {
	session, err := mgo.DialWithInfo(globals.MongoDialInfo)
	if err != nil {
		return err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	collection := session.DB(conf.Mongo.Database).C(name)
	query := collection.Find(criteria)
	if projection != nil {
		log.Println("Add projection in mongo query")
		query = query.Select(projection)
	}
	log.Println("before find")
	return query.One(result)
}

func upsertMongoCollection(name string, selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error) {
	session, err := mgo.DialWithInfo(globals.MongoDialInfo)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	collection := session.DB(conf.Mongo.Database).C(name)
	return collection.Upsert(selector, update)
}

func updateMongoCollection(name string, selector interface{}, update interface{}) (err error) {
	session, err := mgo.DialWithInfo(globals.MongoDialInfo)
	if err != nil {
		return err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	collection := session.DB(conf.Mongo.Database).C(name)
	return collection.Update(selector, update)
}
