package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kateshostak/grader/internal/pkg/auth"
	"github.com/kateshostak/grader/internal/pkg/session"
	tasksrepo "github.com/kateshostak/grader/internal/pkg/tasks"
)

type handlerWithUser func(w http.ResponseWriter, r *http.Request, user *tasksrepo.User)

func Auth(auth auth.Auth, sessionManager session.Sessioner, tasksDB tasksrepo.Tasker, handler handlerWithUser) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		authHeader := r.Header.Get("Authorization")
		if len(authHeader) == 0 {
			http.Error(w, "no token was provided in header", http.StatusInternalServerError)
			return
		}

		authParts := strings.Split(authHeader, " ")
		if len(authParts) < 2 {
			http.Error(w, "auth header field is in invalid - either no method or token is provided", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ExtractClaims(authParts[1])
		if err != nil {
			http.Error(w, fmt.Sprintf("token is invalid or expired: %v", err), http.StatusUnauthorized)
			return
		}

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
		}

		handler(w, r, dbUser)
	})
}
