package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kateshostak/grader/internal/pkg/auth"
	"github.com/kateshostak/grader/internal/pkg/session"
	tasksrepo "github.com/kateshostak/grader/internal/pkg/tasks"
)

func Auth(auth auth.Auth, sessionManager session.Sessioner, tasksDB tasksrepo.Tasker, handler handlerWithUser) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtStr, err := r.Cookie("grader")
		if err != nil {
			http.Redirect(w, r, "/login", 302)
			return
		}

		claims, err := auth.ExtractClaims(jwtStr.Value)
		if err != nil {
			http.Redirect(w, r, "/login", 302)
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		if !sessionManager.IsValid(ctx, strconv.FormatUint(claims.User.ID, 10), claims.StandardClaims.Id) {
			http.Error(w, "session is invalid", http.StatusUnauthorized)
			return
		}

		dbUser, err := tasksDB.GetUserById(ctx, claims.User.ID)
		if err != nil {
			if err == tasksrepo.ErrNoUser {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {

				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		fmt.Println()
		handler(w, r, dbUser)
	})
}
