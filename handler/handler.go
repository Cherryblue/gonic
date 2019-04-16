package handler

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gorilla/sessions"
	"github.com/sentriz/gonic/db"
	"github.com/sentriz/gonic/subsonic"

	"github.com/jinzhu/gorm"
	"github.com/wader/gormstore"
)

var (
	templates = make(map[string]*template.Template)
)

func init() {
	templates["login"] = template.Must(template.ParseFiles(
		filepath.Join("templates", "layout.tmpl"),
		filepath.Join("templates", "pages", "login.tmpl"),
	))
	templates["home"] = template.Must(template.ParseFiles(
		filepath.Join("templates", "layout.tmpl"),
		filepath.Join("templates", "user.tmpl"),
		filepath.Join("templates", "pages", "home.tmpl"),
	))
	templates["create_user"] = template.Must(template.ParseFiles(
		filepath.Join("templates", "layout.tmpl"),
		filepath.Join("templates", "user.tmpl"),
		filepath.Join("templates", "pages", "create_user.tmpl"),
	))
	templates["change_password"] = template.Must(template.ParseFiles(
		filepath.Join("templates", "layout.tmpl"),
		filepath.Join("templates", "user.tmpl"),
		filepath.Join("templates", "pages", "change_password.tmpl"),
	))
}

type Controller struct {
	DB        *gorm.DB
	SStore    *gormstore.Store
	Templates map[string]*template.Template
}

type templateData struct {
	Flashes      []interface{}
	User         *db.User
	SelectedUser *db.User
	AllUsers     []*db.User
	ArtistCount  uint
	AlbumCount   uint
	TrackCount   uint
}

func getStrParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func getIntParam(r *http.Request, key string) (int, error) {
	strVal := r.URL.Query().Get(key)
	if strVal == "" {
		return 0, fmt.Errorf("no param with key `%s`", key)
	}
	val, err := strconv.Atoi(strVal)
	if err != nil {
		return 0, fmt.Errorf("not an int `%s`", strVal)
	}
	return val, nil
}

func getIntParamOr(r *http.Request, key string, or int) int {
	val, err := getIntParam(r, key)
	if err != nil {
		return or
	}
	return val
}

func respondRaw(w http.ResponseWriter, r *http.Request,
	code int, sub *subsonic.Response) {
	res := subsonic.MetaResponse{
		Response: sub,
	}
	switch r.URL.Query().Get("f") {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(res)
		if err != nil {
			log.Printf("could not marshall to json: %v\n", err)
		}
		w.Write(data)
	case "jsonp":
		w.Header().Set("Content-Type", "application/javascript")
		data, err := json.Marshal(res)
		if err != nil {
			log.Printf("could not marshall to json: %v\n", err)
		}
		callback := r.URL.Query().Get("callback")
		w.Write([]byte(callback))
		w.Write([]byte("("))
		w.Write(data)
		w.Write([]byte(");"))
	default:
		w.Header().Set("Content-Type", "application/xml")
		data, err := xml.Marshal(res)
		if err != nil {
			log.Printf("could not marshall to xml: %v\n", err)
		}
		w.Write(data)
	}
}

func respond(w http.ResponseWriter, r *http.Request,
	sub *subsonic.Response) {
	respondRaw(w, r, http.StatusOK, sub)
}

func respondError(w http.ResponseWriter, r *http.Request,
	code uint64, message string) {
	respondRaw(w, r, http.StatusBadRequest, subsonic.NewError(
		code, message,
	))
}

func renderTemplate(w http.ResponseWriter, r *http.Request,
	s *sessions.Session, name string, data *templateData) {
	// take the flashes from the session and add to template
	data.Flashes = s.Flashes()
	s.Save(r, w)
	// take the user gob from the session (if we're logged in and
	// it's there) cast to a user and add to the template
	userIntf := s.Values["user"]
	if userIntf != nil {
		data.User = s.Values["user"].(*db.User)
	}
	err := templates[name].ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("500 when executing: %v", err), 500)
		return
	}
}
