package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"text/template"
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
		return nil, fmt.Errorf("can't connect to redis: %v", err)
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

	grader.router.HandleFunc("/task/{taskID}",
		middleware.Auth(
			grader.auth,
			grader.sessionManager,
			grader.tasks,
			grader.ShowTask)).Methods("GET")

	grader.router.HandleFunc("/task/{taskID}",
		middleware.Auth(
			grader.auth,
			grader.sessionManager,
			grader.tasks,
			grader.SubmitSolution)).Methods("POST")

	grader.router.HandleFunc("/create",
		middleware.Auth(
			grader.auth,
			grader.sessionManager,
			grader.tasks,
			middleware.Admin(grader.CreateTask)))

	grader.router.HandleFunc("/login", grader.Login)
	grader.router.HandleFunc("/signup", grader.Signup)

	grader.router.HandleFunc("/", grader.HomePage)
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
			ID:          task.ID,
			Name:        task.Name,
			Description: task.Description,
		})
	}

	tmpl, err := template.ParseFiles("../web/template/tasks.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, res)
}

func (g Grader) CreateTask(w http.ResponseWriter, r *http.Request, user *tasksrepo.User) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("../web/template/create.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("cant parse form %v", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	if err := g.tasks.CreateTask(ctx, &tasksrepo.Task{Name: r.FormValue("name"), Description: r.FormValue("description")}); err != nil {
		http.Error(w, fmt.Sprintf("cant create task:%v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tasks", 302)
}

func (g Grader) ShowTask(w http.ResponseWriter, r *http.Request, user *tasksrepo.User) {
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

	tmpl, err := template.ParseFiles("../web/template/solution.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, task)
}

func (g Grader) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {

		tmpl, err := template.ParseFiles("../web/template/signup.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("cant parse body %v", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	savedUser, err := g.tasks.CreateUser(ctx, &tasksrepo.User{Name: r.FormValue("name"), Password: r.FormValue("password")})
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

	http.SetCookie(w, &http.Cookie{Name: "grader", Value: jwt})
	http.Redirect(w, r, "/tasks", 302)

}

func (g Grader) HomePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("../web/template/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)
}

func (g Grader) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		//should check for cookie and expiration if ok - redirect to tasks
		tmpl, err := template.ParseFiles("../web/template/login.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	savedUser, err := g.tasks.GetUserByName(ctx, r.FormValue("name"))

	if err != nil {
		if err == tasksrepo.ErrNoUser {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		http.Error(w, fmt.Sprintf("can't get user with name:%v, %v", r.FormValue("name"), err), http.StatusInternalServerError)
		return
	}

	if r.FormValue("password") != savedUser.Password {
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

	http.SetCookie(w, &http.Cookie{Name: "grader", Value: jwt})
	http.Redirect(w, r, "/tasks", 302)
}

func (g Grader) SubmitSolution(w http.ResponseWriter, r *http.Request, user *tasksrepo.User) {
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	taskID, err := strconv.ParseUint(mux.Vars(r)["taskID"], 10, 32)
	if err != nil {
		http.Error(w, fmt.Sprint("invalid task id: %v", err), http.StatusUnprocessableEntity)
		return
	}

	if _, err := g.tasks.GetTaskByID(ctx, uint32(taskID)); err != nil {
		if err == tasksrepo.ErrNoTask {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("can't parse form, taskID:%v, userID:%v, err %v", taskID, user.ID, err), http.StatusInternalServerError)
		return
	}

	if err := g.queue.Add(ctx, queuerepo.Solution{
		TaskID:    uint32(taskID),
		UserID:    user.ID,
		CreatedAt: time.Now(),
		Solution:  []byte(r.FormValue("solution")),
		Status:    "New",
	}); err != nil {
		http.Error(w, fmt.Sprintf("can't submit solution for userID %v, taskID %v, err %v", user.ID, taskID, err), http.StatusInternalServerError)
		return
	}

	//add message to queue
	fmt.Println("success")
}
