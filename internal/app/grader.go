package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	tasksrepo "github.com/kateshostak/grader/internal/pkg/tasks"
	tasks "github.com/kateshostak/grader/internal/pkg/tasks/db"
)

const timeout = time.Second

type taskJSON struct {
	ID          uint32
	Name        string
	Description string
}

type userJSON struct {
	ID       uint64
	Name     string
	Password string
}

type Grader struct {
	tasks  tasksrepo.Tasker
	router *mux.Router
}

func (g *Grader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.router.ServeHTTP(w, r)
}

func NewGrader(tasksURL string) (*Grader, error) {
	tasksDB, err := createPostgresDB(tasksURL)
	if err != nil {
		return nil, err
	}
	tasker := tasks.NewTasker(tasksDB)
	return createGrader(tasker), nil
}

func createGrader(tasks tasksrepo.Tasker) *Grader {
	router := mux.NewRouter()
	grader := &Grader{
		tasks:  tasks,
		router: router,
	}

	grader.router.HandleFunc("/tasks", grader.ListAllTasks).Methods("GET")
	grader.router.HandleFunc("/task/{taskID}", grader.GetTaskByID).Methods("GET")

	//grader.router.HandleFunc("/create", grader.CreateTask).Methods("GET")
	grader.router.HandleFunc("/create", grader.CreateTask).Methods("POST")

	//grader.router.HandleFunc("/login", grader.Login).Methods("GET")
	//grader.router.HandleFunc("/login", grader.Login).Methods("POST")

	//	grader.router.HandleFunc("/signup", grader.Signup).Methods("GET")
	//grader.router.HandleFunc("/signup", grader.Signup).Methods("POST")

	return grader
}

func createPostgresDB(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("cant connect to postgres: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("cant ping url: %v, %v", dbURL, err)
	}

	return db, nil
}

func (g *Grader) ListAllTasks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	tasksSlice, err := g.tasks.GetAllTasks(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("cant get tasks: %v", err), http.StatusInternalServerError)
		return
	}

	res := []taskJSON{}
	for _, task := range tasksSlice {
		res = append(res, taskJSON{
			ID:   task.ID,
			Name: task.Name,
		})
	}

	json.NewEncoder(w).Encode(res)
}

func (g *Grader) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	var task taskJSON

	err := json.NewDecoder(r.Body).Decode(&task)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, fmt.Sprintf("cant parse body %v", err), http.StatusInternalServerError)
		return
	}

	if err := g.tasks.CreateTask(ctx, &tasksrepo.Task{ID: task.ID, Name: task.Name, Description: task.Description}); err != nil {
		http.Error(w, fmt.Sprintf("cant create task:%v", err), http.StatusInternalServerError)
		return
	}
}

func (g *Grader) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	id, err := strconv.Atoi(mux.Vars(r)["taskID"])
	if err != nil {
		http.Error(w, fmt.Sprint("invalid task id: %v", err), http.StatusUnprocessableEntity)
	}

	task, err := g.tasks.GetTaskByID(ctx, uint32(id))
	if err != nil {
		if err == tasksrepo.ErrNoTask {
			http.Error(w, fmt.Sprintf("cant get task with given id:%v, %v", id, err), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("cant get task with given id:%v, %v", id, err), http.StatusNotFound)
			return
		}
	}
	json.NewEncoder(w).Encode(taskJSON{ID: task.ID, Name: task.Name, Description: task.Description})
}
