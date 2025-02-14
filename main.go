package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"

	mgo "gopkg.in/mgo.v2"
)

var rnd *renderer.Render

// Render is a Go package that helps simplify the process of writing JSON, HTML, or other responses in web applications.
var db *mgo.Database

const (
	hostName       string = "localhost:2107"
	dbName         string = "TrekDB"
	collectionName string = "trek"
	port           string = ":9000"
)

func init() {
	sess, err := mgo.Dial(hostName)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
	db = sess.DB(dbName)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func homeHandler(res http.ResponseWriter, req *http.Request) {
	err := rnd.Template(res, http.StatusOK, []string{"static/home.tpl"}, nil)
	checkErr(err)
}

func main() {
	//creating channel
	server := make(chan os.Signal)
	signal.Notify(server, os.Interrupt)
	fmt.Println("Setup done")

	//creating router
	// refers to a component that manages how incoming HTTP requests are directed to specific handlers
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/", homeHandler)

	//stop the server gresfully
	<-server
	log.Println("Shutting down server...")

}
