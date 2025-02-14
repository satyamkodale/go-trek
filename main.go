package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type (
	todoEntity struct {
		ID bson.ObjectId `bson:"_id,omitempty"`
		// the content inside (``) show how fields are  going to stored in the db
		Title     string    `bson:"title"`
		Completed bool      `bson:"completed"`
		CreatedAt time.Time `bson:"createdAt"`
	}
	todoDTO struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Completed bool      `json:"completed"`
		CreatedAt time.Time `json:"created_at"`
	}
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

func getAllTodo(res http.ResponseWriter, req *http.Request) {
	todoEntities := []todoEntity{}

	//get the enitited from db
	if err := db.C(collectionName).Find(bson.M{}).All(&todoEntities); err != nil {
		rnd.JSON(res, http.StatusInternalServerError, renderer.M{
			"message": "Failed to fetch todo",
			"error":   err,
		})
		return
	}

	//convert into JSON DTO's
	todoList := []todoDTO{}

	// ID        string    `json:"id"`
	// 	Title     string    `json:"title"`
	// 	Completed bool      `json:"completed"`
	// 	CreatedAt time.Time `json:"created_at"`

	for _, todo := range todoEntities {
		todoList = append(todoList, todoDTO{
			ID:        todo.ID.Hex(), // Convert BSON ObjectId to string
			Title:     todo.Title,
			Completed: todo.Completed,
			CreatedAt: todo.CreatedAt,
		})
	}

	rnd.JSON(res, http.StatusOK, renderer.M{
		"data": todoList,
	})

}

func deleteTodo(res http.ResponseWriter, req *http.Request) {
	id := strings.TrimSpace(chi.URLParam(req, "id"))
	// if id not found
	if !bson.IsObjectIdHex(id) {
		rnd.JSON(res, http.StatusBadRequest, renderer.M{"Message": "Id not found"})
		return
	}
	if err := db.C(collectionName).RemoveId(bson.ObjectIdHex(id)); err != nil {
		rnd.JSON(res, http.StatusProcessing, renderer.M{
			"message": "Failed to delete todo",
			"error":   err,
		})
		return
	}
	rnd.JSON(res, http.StatusOK, renderer.M{
		"message": "Todo deleted successfully",
	})

}

func createTodo(res http.ResponseWriter, req *http.Request) {
	var todo todoDTO

	if err := json.NewDecoder(req.Body).Decode(&todo); err != nil {
		rnd.JSON(res, http.StatusProcessing, err)
		return
	}

	//validation
	// simple validation
	if todo.Title == "" {
		rnd.JSON(res, http.StatusBadRequest, renderer.M{
			"message": "The title field is requried",
		})
		return
	}

	todoEntity := todoEntity{
		ID:        bson.NewObjectId(),
		Title:     todo.Title,
		Completed: false,
		CreatedAt: time.Now(),
	}

	if err := db.C(collectionName).Insert(&todoEntity); err != nil {
		rnd.JSON(res, http.StatusInternalServerError, renderer.M{
			"message": "Failed to save todo",
			"error":   err,
		})
		return
	}

	rnd.JSON(res, http.StatusCreated, renderer.M{
		"message": "Todo created successfully",
		"todo_id": todoEntity.ID.Hex(),
	})

}

func updateTodo(res http.ResponseWriter, req *http.Request) {
	id := strings.TrimSpace(chi.URLParam(req, "id"))
	// if id not found
	if !bson.IsObjectIdHex(id) {
		rnd.JSON(res, http.StatusBadRequest, renderer.M{
			"Message": "Id not found",
		})
		return
	}

	// extract the dto from req
	var todoDTO todoDTO
	if err := json.NewDecoder(req.Body).Decode(&todoDTO); err != nil {
		rnd.JSON(res, http.StatusProcessing, err)
		return
	}

	// validation
	if todoDTO.Title == "" {
		rnd.JSON(res, http.StatusBadRequest, renderer.M{
			"message": "The title field is requried",
		})
		return
	}

	//update todo
	if err := db.C(collectionName).
		Update(
			bson.M{"_id": bson.ObjectIdHex(id)},
			bson.M{"title": todoDTO.Title, "completed": todoDTO.Completed},
		); err != nil {
		rnd.JSON(res, http.StatusProcessing, renderer.M{
			"message": "Failed to update todo",
			"error":   err,
		})
		return
	}

	rnd.JSON(res, http.StatusOK, renderer.M{
		"message": "Todo updated successfully",
	})

}

// we don't use parentheses (e.g., getAllTodo()) when passing these functions as arguments is because we're passing the function references, not calling them immediately.
func todoHandlers() http.Handler {
	router := chi.NewRouter()
	router.Get("/", getAllTodo)
	router.Delete("/{id}", deleteTodo)
	router.Post("/", createTodo)
	router.Put("/{id}", updateTodo)

	return router
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
	router.Mount("/todo", todoHandlers())

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
