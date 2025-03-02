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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

var rend *renderer.Render
var mongoClient *mongo.Client

const (
	port string = ":9999"
)

type (
	todoEntity struct {
		ID        primitive.ObjectID `bson:"_id,omitempty"`
		Title     string             `bson:"title"`
		Completed bool               `bson:"completed"`
		CreatedAt time.Time          `bson:"createdAt"`
	}
	todoDTO struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Completed bool      `json:"completed"`
		CreatedAt time.Time `json:"created_at"`
	}
)

func test(res http.ResponseWriter, req *http.Request) {
	rend.JSON(res, http.StatusOK, renderer.M{
		"message": "application running",
	})
}

func createTodo(res http.ResponseWriter, req *http.Request) {
	var todo todoDTO

	// Decode JSON request
	if err := json.NewDecoder(req.Body).Decode(&todo); err != nil {
		rend.JSON(res, http.StatusBadRequest, map[string]interface{}{
			"message": "Invalid request payload",
			"error":   err.Error(),
		})
		return
	}

	// Simple validation
	if todo.Title == "" {
		rend.JSON(res, http.StatusBadRequest, map[string]interface{}{
			"message": "The title field is required",
		})
		return
	}

	// Create a new todo entity
	todoEntity := todoEntity{
		ID:        primitive.NewObjectID(),
		Title:     todo.Title,
		Completed: todo.Completed,
		CreatedAt: time.Now(),
	}

	// Insert into MongoDB
	_, err := mongoClient.Database("TrekDB").Collection("trek").InsertOne(context.Background(), todoEntity)
	if err != nil {
		rend.JSON(res, http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed to save todo",
			"error":   err.Error(),
		})
		return
	}

	// Success response
	rend.JSON(res, http.StatusCreated, map[string]interface{}{
		"message": "Todo created successfully",
		"todo_id": todoEntity.ID.Hex(),
	})

}
func deleteTodo(res http.ResponseWriter, req *http.Request) {
	id := strings.TrimSpace(chi.URLParam(req, "id"))

	// Validate ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		rend.JSON(res, http.StatusBadRequest, map[string]interface{}{
			"message": "Invalid Object ID",
		})
		return
	}

	// Delete document
	result, err := mongoClient.Database("TrekDB").Collection("trek").DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		rend.JSON(res, http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed to delete todo",
			"error":   err.Error(),
		})
		return
	}

	// Check if a document was actually deleted
	if result.DeletedCount == 0 {
		rend.JSON(res, http.StatusNotFound, map[string]interface{}{
			"message": "Todo not found",
		})
		return
	}

	// Success response
	rend.JSON(res, http.StatusOK, map[string]interface{}{
		"message": "Todo deleted successfully",
	})
}

func updateTodo(res http.ResponseWriter, req *http.Request) {
	id := strings.TrimSpace(chi.URLParam(req, "id"))

	// Validate ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		rend.JSON(res, http.StatusBadRequest, map[string]interface{}{
			"message": "Invalid Object ID",
		})
		return
	}

	// Extract DTO from request
	var todoDTO todoDTO
	if err := json.NewDecoder(req.Body).Decode(&todoDTO); err != nil {
		rend.JSON(res, http.StatusBadRequest, map[string]interface{}{
			"message": "Invalid request body",
			"error":   err.Error(),
		})
		return
	}

	// Validation
	if todoDTO.Title == "" {
		rend.JSON(res, http.StatusBadRequest, map[string]interface{}{
			"message": "The title field is required",
		})
		return
	}

	// Prepare update data
	update := bson.M{
		"$set": bson.M{
			"title":     todoDTO.Title,
			"completed": todoDTO.Completed,
		},
	}

	// Update todo in MongoDB
	result, err := mongoClient.Database("TrekDB").Collection("trek").UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		update,
	)

	if err != nil {
		rend.JSON(res, http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed to update todo",
			"error":   err.Error(),
		})
		return
	}

	// Check if the document was actually updated
	if result.MatchedCount == 0 {
		rend.JSON(res, http.StatusNotFound, map[string]interface{}{
			"message": "Todo not found",
		})
		return
	}

	rend.JSON(res, http.StatusOK, map[string]interface{}{
		"message": "Todo updated successfully",
	})
}

func getAllTodo(res http.ResponseWriter, req *http.Request) {
	todoEntities := []todoEntity{}
	collection := mongoClient.Database("TrekDB").Collection("trek")
	cursor, err := collection.Find(context.Background(), bson.M{})

	if err != nil {
		http.Error(res, "Failed to fetch todos", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &todoEntities); err != nil {
		http.Error(res, "Failed to decode todos", http.StatusInternalServerError)
		return
	}
	todoDTOs := []todoDTO{}
	for _, todo := range todoEntities {
		todoDTOs = append(todoDTOs, todoDTO{
			ID:        todo.ID.Hex(),
			Title:     todo.Title,
			Completed: todo.Completed,
			CreatedAt: todo.CreatedAt,
		})
	}

	rend.JSON(res, http.StatusOK, renderer.M{
		"data": todoDTOs,
	})
}

func homeHandler(res http.ResponseWriter, req *http.Request) {
	err := rend.Template(res, http.StatusOK, []string{"static/home.tpl"}, nil)
	checkErr(err)
}

func todoRoutes() http.Handler {
	router := chi.NewRouter()
	router.Get("/", getAllTodo)
	router.Post("/", createTodo)
	router.Delete("/{id}", deleteTodo)
	router.Put("/{id}", updateTodo)
	return router
}

func main() {
	channel := make(chan os.Signal, 1) // buffered channel
	signal.Notify(channel, os.Interrupt)
	fmt.Println("Setup done")
	fmt.Println("WELCOME TO GO_TREK")

	// Initialize renderer
	rend = renderer.New()

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	mongoClient = client //store the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatalf("Error pinging MongoDB: %v", err)
	}
	fmt.Println("Connected to DB")

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/test", test)
	router.Get("/", homeHandler)
	router.Mount("/todo", todoRoutes())

	server := &http.Server{
		Addr:         port,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Println("Listening port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Graceful shutdown
	<-channel
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped!")

	// Disconnect from MongoDB
	if err := mongoClient.Disconnect(context.TODO()); err != nil {
		log.Fatalf("Error disconnecting from MongoDB: %v", err)
	}
	log.Println("Disconnected from MongoDB")
}

func checkErr(err error) {
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
