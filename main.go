package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

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
	channel := make(chan os.Signal)
	signal.Notify(channel, os.Interrupt)
	fmt.Println("Setup done")

	//creating router
	// refers to a component that manages how incoming HTTP requests are directed to specific handlers
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/", homeHandler)

	//creating the server
	server := &http.Server{
		Addr:         port,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		fmt.Println("Listening port", port)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	//stop the server gresfully
	<-channel
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	server.Shutdown(ctx)
	defer cancel()
	log.Println("Server gracefully stopped!")

}
