package main

import (
	"fmt"
	"github.com/albrow/forms"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/martini-contrib/cors"
	"github.com/unrolled/render"
	"net/http"
	"strconv"
)

// NOTE: This is a test server specifically designed for testing the humble framework.
// As such, it is designed to be completely idempotent. That means nothing you do will
// actually change the data on the server, and sending the same request will always
// give you the same response. However, when possible the responses are designed to mimic
// that of a real server that does hold state.

type todo struct {
	Id          int
	Title       string
	IsCompleted bool
}

// Since the server is idempotent, the list of todos will never change, regardless of
// requests to create, update, or delete todos.
var todos = []todo{
	{
		Id:          0,
		Title:       "Todo 0",
		IsCompleted: false,
	},
	{
		Id:          1,
		Title:       "Todo 1",
		IsCompleted: false,
	},
	{
		Id:          2,
		Title:       "Todo 2",
		IsCompleted: true,
	},
}

var (
	// r is used to render responses
	r = render.New(render.Options{
		IndentJSON: true,
	})
)

const (
	statusUnprocessableEntity = 422
)

func main() {
	// Routes
	router := mux.NewRouter()
	router.HandleFunc("/todos", todosController.Index).Methods("GET")
	router.HandleFunc("/todos", todosController.Create).Methods("POST")
	router.HandleFunc("/todos/{id}", todosController.Show).Methods("GET")
	router.HandleFunc("/todos/{id}", todosController.Update).Methods("PUT")
	router.HandleFunc("/todos/{id}", todosController.Delete).Methods("DELETE")

	// Other middleware
	n := negroni.New(negroni.NewLogger())
	n.UseHandler(cors.Allow(&cors.Options{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "PUT", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Router must always come last
	n.UseHandler(router)

	// Start the server
	n.Run(":3000")
}

// Todos Controller and its methods
type todosControllerType struct{}

var todosController = todosControllerType{}

// Index returns a list of todos as an array of json objects. It always returns the
// same list of todos and is idempotent.
func (todosControllerType) Index(w http.ResponseWriter, req *http.Request) {
	r.JSON(w, http.StatusOK, todos)
}

// Create accepts form data for creating a new todo. Since this server is designed
// for testing, it does not actually create the todo, as that would make the server
// non-idempotent. Create returns the todo that would be created as a json object.
// It assigns the id of 3 to the todo.
func (todosControllerType) Create(w http.ResponseWriter, req *http.Request) {
	// Parse data and do validations
	todoData, err := forms.Parse(req)
	if err != nil {
		panic(err)
	}
	val := todoData.Validator()
	val.Require("Title")
	val.Require("IsCompleted")
	val.TypeBool("IsCompleted")
	if val.HasErrors() {
		r.JSON(w, statusUnprocessableEntity, val.ErrorMap())
		return
	}

	// Return the todo that would be created
	todo := todo{
		Id:          3,
		Title:       todoData.Get("Title"),
		IsCompleted: todoData.GetBool("IsCompleted"),
	}
	r.JSON(w, http.StatusOK, todo)
}

// Show returns the json data for an existing todo. Since the todos never change
// and there are three of them, Show will only respond with a todo object for id
// parameters between 0 and 2. Any other id will result in a 422 error.
func (todosControllerType) Show(w http.ResponseWriter, req *http.Request) {
	// Get the id from the url parameters
	id, err := parseId(req)
	if err != nil {
		r.JSON(w, statusUnprocessableEntity, map[string]error{
			"error": err,
		})
		return
	}
	r.JSON(w, http.StatusOK, todos[id])
}

func (todosControllerType) Update(w http.ResponseWriter, req *http.Request) {
	// Get the id from the url parameters
	id, err := parseId(req)
	if err != nil {
		r.JSON(w, statusUnprocessableEntity, map[string]error{
			"error": err,
		})
		return
	}
	// Create a copy of the todo corresponding to id
	todoCopy := todos[id]
	// Parse data from the request
	todoData, err := forms.Parse(req)
	if err != nil {
		panic(err)
	}
	// Validate and update the data only if it was provided in the request
	if todoData.KeyExists("IsCompleted") {
		val := todoData.Validator()
		val.TypeBool("IsCompleted")
		if val.HasErrors() {
			r.JSON(w, statusUnprocessableEntity, val.ErrorMap())
			return
		}
		// Update todoCopy with the given data
		todoCopy.IsCompleted = todoData.GetBool("IsCompleted")
	}
	if todoData.KeyExists("Title") {
		todoCopy.Title = todoData.Get("Title")
	}
	r.JSON(w, http.StatusOK, todoCopy)
}

func (todosControllerType) Delete(w http.ResponseWriter, req *http.Request) {
	// Get the id from the url parameters
	if _, err := parseId(req); err != nil {
		r.JSON(w, statusUnprocessableEntity, map[string]error{
			"error": err,
		})
		return
	}
	r.JSON(w, http.StatusOK, struct{}{})
}

// parseId gets the id out of the url parameters of req, converts it to an int,
// and then checks that it is in the range of existing todos. It will return an
// an error if there was problem converting the id parameter to an int or the
// id was outside the range of existing todos.
func parseId(req *http.Request) (int, error) {
	idStr := mux.Vars(req)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf(`Could not convert id paramater "%s" to int`, idStr)
	}
	if id < 0 || id > 2 {
		return 0, fmt.Errorf(`Could not find todo with id = %d`, id)
	}
	return id, nil
}
