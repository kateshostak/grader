package middleware

import (
	"net/http"

	tasksrepo "github.com/kateshostak/grader/internal/pkg/tasks"
)

type handlerWithUser func(w http.ResponseWriter, r *http.Request, user *tasksrepo.User)

func Admin(handler handlerWithUser) handlerWithUser {
	return func(w http.ResponseWriter, r *http.Request, user *tasksrepo.User) {
		if !user.IsAdmin {
			http.Error(w, "Only admin can create tasks", http.StatusUnauthorized)
			http.Redirect(w, r, "/", 302)
			return
		}
		handler(w, r, user)
	}
}
