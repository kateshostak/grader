package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/gorilla/mux"
	"github.com/kateshostak/grader/internal/pkg/auth"
	middleware "github.com/kateshostak/grader/internal/pkg/middleware"
	queuerepo "github.com/kateshostak/grader/internal/pkg/queue"
	queue "github.com/kateshostak/grader/internal/pkg/queue/db"
	"github.com/kateshostak/grader/internal/pkg/session"
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
	tasks          tasksrepo.Tasker
	queue          queuerepo.Queuer
	auth           auth.Auth
	sessionManager session.Sessioner
	router         *mux.Router
}

func (g Grader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.router.ServeHTTP(w, r)
}

func NewGrader(tasksURL, queueURL, sessionURL string) (*Grader, error) {
	tasksDB, err := createPostgresDB(tasksURL)
	if err != nil {
		return nil, err
	}
	tasker := tasks.NewTasker(tasksDB)

	queueDB, err := createPostgresDB(queueURL)
	if err != nil {
		return nil, err
	}
	queuer := queue.NewQueuer(queueDB)

	redisDB, err := createRedisDB(sessionURL)
	if err != nil {
		return nil, err
	}

	return createGrader(tasker, queuer, auth.NewAuth(), session.NewManager(redisDB)), nil
}

func createRedisDB(redisURL string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("can's connect to redis: %v", err)
	}

	return client, nil
}

func createGrader(tasks tasksrepo.Tasker, queue queuerepo.Queuer, auth auth.Auth, sessionManager session.Sessioner) *Grader {
	router := mux.NewRouter()
	grader := &Grader{
		tasks:          tasks,
		queue:          queue,
		auth:           auth,
		sessionManager: sessionManager,
		router:         router,
	}

	grader.router.HandleFunc("/tasks", grader.ListAllTasks).Methods("GET")
	grader.router.HandleFunc("/task/{taskID}", grader.GetTaskByID).Methods("GET")
	grader.router.HandleFunc("/task/{taskID}", middleware.Auth(grader.auth, grader.sessionManager, grader.tasks, grader.SubmitSolution)).Methods("POST")

	//grader.router.HandleFunc("/create", grader.CreateTask).Methods("GET")
	grader.router.HandleFunc("/create", middleware.Auth(grader.auth, grader.sessionManager, grader.tasks, grader.CreateTask)).Methods("POST")

	//grader.router.HandleFunc("/login", grader.Login).Methods("GET")
	grader.router.HandleFunc("/login", grader.Login).Methods("POST")

	//	grader.router.HandleFunc("/signup", grader.Signup).Methods("GET")
	grader.router.HandleFunc("/signup", grader.Signup).Methods("POST")

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

func (g Grader) ListAllTasks(w http.ResponseWriter, r *http.Request) {
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

//authmiddleware + adminmiddleware
func (g Grader) CreateTask(w http.ResponseWriter, r *http.Request, user *tasksrepo.User) {
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
	//redirect to list all tasks
}

func (g Grader) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	id, err := strconv.ParseUint(mux.Vars(r)["taskID"], 10, 32)
	if err != nil {
		http.Error(w, fmt.Sprint("invalid task id: %v", err), http.StatusUnprocessableEntity)
		return
	}

	task, err := g.tasks.GetTaskByID(ctx, uint32(id))
	if err != nil {
		if err == tasksrepo.ErrNoTask {
			http.Error(w, fmt.Sprintf("cant get task with given id:%v, %v", id, err), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("cant get task with given id:%v, %v", id, err), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(taskJSON{ID: task.ID, Name: task.Name, Description: task.Description})
}

func (g Grader) Signup(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	var user userJSON

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, fmt.Sprintf("cant parse body %v", err), http.StatusInternalServerError)
		return
	}

	savedUser, err := g.tasks.CreateUser(ctx, &tasksrepo.User{Name: user.Name, Password: user.Password})
	if err != nil {
		http.Error(w, fmt.Sprintf("cant create user:%v", err), http.StatusInternalServerError)
		return
	}

	ttl := 30 * time.Minute
	issuedAt := time.Now().Truncate(time.Second)
	jwt, jti, err := g.auth.GetSignedToken(auth.User{ID: savedUser.ID, Name: savedUser.Name}, issuedAt, ttl)

	if err != nil {
		http.Error(w, fmt.Sprintf("can't get jwt for user with id: %v, %v", savedUser.ID, err), http.StatusInternalServerError)
		return
	}

	if err := g.sessionManager.Add(ctx, strconv.FormatUint(savedUser.ID, 10), jti, issuedAt.Add(ttl)); err != nil {
		return
	}

	if err := json.NewEncoder(w).Encode(struct{ Token string }{Token: jwt}); err != nil {
		http.Error(w, fmt.Sprintf("can't write response for user with id: %v, %v", savedUser.ID, err), http.StatusInternalServerError)
		return
	}
}

func (g Grader) Login(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	var user userJSON

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "can't decode body", http.StatusInternalServerError)
		return
	}

	savedUser, err := g.tasks.GetUserByName(ctx, user.Name)
	if err != nil {
		if err == tasksrepo.ErrNoUser {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		http.Error(w, fmt.Sprintf("can't get user with name:%v, %v", user.Name, err), http.StatusInternalServerError)
		return
	}

	if user.Password != savedUser.Password {
		http.Error(w, "username or password is invalid", http.StatusBadRequest)
		return
	}

	issuedAt := time.Now().Truncate(time.Second)
	ttl := 30 * time.Minute
	jwt, jti, err := g.auth.GetSignedToken(auth.User{ID: savedUser.ID, Name: savedUser.Name}, issuedAt, ttl)

	if err != nil {
		http.Error(w, fmt.Sprintf("can't get jwt for user with id: %v, %v", savedUser.ID, err), http.StatusInternalServerError)
		return
	}

	if err := g.sessionManager.Add(ctx, strconv.FormatUint(savedUser.ID, 10), jti, issuedAt.Add(ttl)); err != nil {
		return
	}

	if err := json.NewEncoder(w).Encode(struct{ Token string }{Token: jwt}); err != nil {
		http.Error(w, fmt.Sprintf("can't write response for user with id: %v, %v", savedUser.ID, err), http.StatusInternalServerError)
		return
	}
}

//auth middleware
func (g Grader) SubmitSolution(w http.ResponseWriter, r *http.Request, user *tasksrepo.User) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	taskID, err := strconv.ParseUint(mux.Vars(r)["taskID"], 10, 32)
	if err != nil {
		http.Error(w, fmt.Sprint("invalid task id: %v", err), http.StatusUnprocessableEntity)
		return
	}

	solution, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, fmt.Sprintf("can't read body, taskID:%v, userID:%v, err %v", taskID, user.ID, err), http.StatusInternalServerError)
		return
	}

	if err := g.queue.Add(ctx, queuerepo.Solution{
		TaskID:   uint32(taskID),
		UserID:   user.ID,
		Solution: solution,
		Status:   "New",
	}); err != nil {
		http.Error(w, fmt.Sprintf("can't submit solution for userID %v, taskID %v, err %v", user.ID, taskID, err), http.StatusInternalServerError)
		return
	}

}
