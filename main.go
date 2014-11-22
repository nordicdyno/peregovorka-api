package main

// TODO :   add statistic on /stat url
//          cleanup code (grep TODO/FIXME/XXX)
//          add flags
//          Add tests
// MAYBE: change topic/channels naming schema (add prefix, or allow real names)

import (
	//"bytes"
	//"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
	"time"
	//"strings"

	//"github.com/bradfitz/gomemcache/memcache"
	"github.com/BurntSushi/toml"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	mgo "gopkg.in/mgo.v2"
)

const packageName = "github.com/nordicdyno/peregovorka-api"
const shaLength = 10

var (
	// addr   = flag.String("addr", ":80", "http service address")
	// assets = flag.String("assets", defaultAssetPath(), "path to assets")
	configFile = flag.String("conf", defaultConfigFile(), "path to assets")

	homeTempl *template.Template
)

type Config struct {
	Bind         string
	ResourcesDir string `toml:"resources_dir"`
	Db           *DbConfig
	Memcache     *MemcacheConfig
	Mongo        *MongoConfig
}

type MemcacheConfig struct {
	ConnStr string `toml:"conn"`
	Attemps int32
}

type DbConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Dbname   string
	Sslmode  string
}

type MongoConfig struct {
	Addrs []string
	//	Timeout  time.Duration
	TimeoutSec int `toml:"timeout_sec"`
	Database   string
	Username   string
	Password   string
}

var conf = Config{
	Db:       &DbConfig{},
	Memcache: &MemcacheConfig{},
	Mongo:    &MongoConfig{},
}

type Globals struct {
	MongoDialInfo *mgo.DialInfo
	MemClient     *memcache.Client
	DbLink *sqlx.DB
	DbConfString string
}

var globals = Globals{
	MongoDialInfo: &mgo.DialInfo{},
}

func init() {
	flag.Parse()
	if _, err := toml.DecodeFile(*configFile, &conf); err != nil {
		log.Fatalf("TOML %s parse error: %s\n", *configFile, err.Error())
	}
	log.Println("rooms result:", spew.Sdump(&conf))

	if len(conf.ResourcesDir) == 0 {
		conf.ResourcesDir = defaultAssetPath()
	}

	globals.MongoDialInfo = &mgo.DialInfo{
		Addrs:    conf.Mongo.Addrs,
		Timeout:  time.Duration(conf.Mongo.TimeoutSec) * time.Second,
		Database: conf.Mongo.Database,
		Username: conf.Mongo.Username,
		Password: conf.Mongo.Password,
	}
	globals.MemClient = memcache.New(conf.Memcache.ConnStr)
	globals.DbConfString = fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		conf.Db.Host, conf.Db.Port, conf.Db.User, conf.Db.Password, conf.Db.Dbname, conf.Db.Sslmode,
	)

	log.Println("MongoDialInfo:", spew.Sdump(globals.MongoDialInfo))
	log.Println("globals.DbConfString =", globals.DbConfString)
	//panic("bye!")
}

func defaultConfigFile() string {
	p, err := build.Default.Import(packageName, "", build.FindOnly)
	if err != nil {
		return "config.toml"
	}
	return filepath.Join(p.Dir, "config.toml")
}

func defaultAssetPath() string {
	p, err := build.Default.Import(packageName, "", build.FindOnly)
	if err != nil {
		return "."
	}
	return filepath.Join(p.Dir, "resources")
}

func homeHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	page, _ := vars["name"]
	homePath := filepath.Join(conf.ResourcesDir, page+".html")
	//log.Println("homePath: " + homePath)
	homeTempl = template.Must(template.ParseFiles(homePath))
	homeTempl.Execute(w, req.Host)
}

func main() {
	var err error
	log.Println("connect db")
	db, err := sqlx.Open("postgres", globals.DbConfString)
	defer db.Close()
	if err != nil {
		//fmt.Printf("Database opening error -->%v\n", err)
		log.Panic("Database error", err)
	}
	err = db.Ping()
	if err != nil {
		//fmt.Printf("Database opening error -->%v\n", err)
		log.Panic("Database ping error", err)
	}
	globals.DbLink = db


	r := mux.NewRouter()

	// production
	r.HandleFunc("/users/register", registerUserHandler)
	r.HandleFunc("/users/list", listUsersHandler)
	r.HandleFunc("/rooms/list", listRoomsHandler)
	// createRoomHandler needs improvement
	r.HandleFunc("/rooms/create", createRoomHandler)
	// in progress
	r.HandleFunc("/rooms/send",   sendRoomHandler)
	r.HandleFunc("/rooms/history", listHistoryRoomsHandler)

	// other
	r.HandleFunc("/page/{name}", homeHandler)

	r.HandleFunc("/users/create_debug", createUserHandlerDebug)
	r.HandleFunc("/rooms/create_debug", createRoomHandlerDebug)

	// r.HandleFunc("/rooms/send/debug", sendRoomHandlerDebug)
	// r.HandleFunc("/rooms/history", historyRoomsHandler)

	h := appHandler{router: r}
	// os.Args = nil // monkeypatch: hide commandline from expvar
	http.Handle("/", h)

	if err = http.ListenAndServe(conf.Bind, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

type appHandler struct {
	router *mux.Router
}

func (h appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Println(rec)
			log.Println("Catch error. Recovering...")

			w.Header().Set("Content-Type", "text/plain;charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			str := fmt.Sprintln(rec)
			io.WriteString(w, str)
		}
	}()
	h.router.ServeHTTP(w, r)
}
