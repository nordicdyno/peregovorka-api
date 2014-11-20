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

	//"github.com/bradfitz/gomemcache/memcache"
	//"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
)

var (
	addr   = flag.String("addr", ":80", "http service address")
	assets = flag.String("assets", defaultAssetPath(), "path to assets")

	homeTempl *template.Template
)

func defaultAssetPath() string {
	p, err := build.Default.Import("github.com/nordicdyno/api-rooms", "", build.FindOnly)
	if err != nil {
		return "."
	}
	return filepath.Join(p.Dir, "resources")
}

func homeHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	page, _ := vars["name"]
	homePath := filepath.Join(*assets, page+".html")
	//log.Println("homePath: " + homePath)
	homeTempl = template.Must(template.ParseFiles(homePath))
	homeTempl.Execute(w, req.Host)
}

func main() {
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/page/{name}", homeHandler)

	r.HandleFunc("/users/add", addUserHandler)
	r.HandleFunc("/users/list", listUsersHandler)

	h := appHandler{router: r}
	// os.Args = nil // monkeypatch: hide commandline from expvar
	http.Handle("/", h)

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

type appHandler struct {
	router *mux.Router
}

func (h appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Println("Catch error. Recovering...")

			w.Header().Set("Content-Type", "text/html;charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			str := fmt.Sprintln(rec)
			io.WriteString(w, str)
		}
	}()
	h.router.ServeHTTP(w, r)
}
