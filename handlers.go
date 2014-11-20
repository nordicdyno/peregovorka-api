package main

import (
	"encoding/json"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
)

const (
	dbName = "chat"
)

var mongoDBDialInfo = mgo.DialInfo{
	Addrs:    []string{"ds047720.mongolab.com:47720"},
	Timeout:  60 * time.Second,
	Database: dbName,
	Username: "sports",
	Password: "hackochat",
}

type UserList []*UserData

type UserData struct {
	Uid       string    `json:"uid"`
	LastLogin time.Time `bson:"last_login_dt" json:"last_login_dt"`
	Online    bool      `json:"online"`
	Name      string    `json:"name"`
	AvatarUrl string    `bson:"avatar_url" json:"avatar_url"`
}

func addUserHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("call addUserHandler")
	uid := req.PostFormValue("uid")
	name := req.PostFormValue("name")
	avatar_url := req.PostFormValue("avatar_url")

	user := UserData{
		Uid:       uid,
		Online:    true,
		Name:      name,
		AvatarUrl: avatar_url,
	}

	selector := bson.M{"uid": user.Uid}
	data := bson.M{"$set": &user}
	res, err := upsertMongo(selector, data)
	if err != nil {
		panic(err)
	}

	log.Println("result:", spew.Sdump(res))
}

func listUsersHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("call listUsersHandler")
	list := make(UserList, 100)
	err := queryMongoAll(nil, &list)
	if err != nil {
		panic(err)
	}
	log.Println(spew.Sdump(&list))

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	jEnc := json.NewEncoder(w)
	jEnc.Encode(&list)
}

func queryMongoAll(query interface{}, result interface{}) error {
	session, err := mgo.DialWithInfo(&mongoDBDialInfo)
	if err != nil {
		return err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	collection := session.DB(dbName).C("users")
	return collection.Find(query).All(result)
}

func upsertMongo(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error) {
	session, err := mgo.DialWithInfo(&mongoDBDialInfo)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	collection := session.DB(dbName).C("users")
	return collection.Upsert(selector, update)
}
