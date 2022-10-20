package main

import (
	"log"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kateshostak/grader/internal/app"
)

func main() {
	grader, err := app.NewGrader("postgres://postgres:password@localhost:5432/tasks", "postgres://postgres:password@localhost:5432/queue")
	if err != nil {
		log.Fatalf("cant create Grader: %v", err)
	}
	//defer grader.Close()

	log.Fatal(http.ListenAndServe("localhost:8080", grader))
}
