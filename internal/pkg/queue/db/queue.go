package queue

import (
	"context"
	"database/sql"

	queuerepo "github.com/kateshostak/grader/internal/pkg/queue"
)

type QueueRepo struct {
	queue *sql.DB
}

func NewQueuer(db *sql.DB) *QueueRepo {
	return &QueueRepo{
		queue: db,
	}
}

func (q QueueRepo) Add(ctx context.Context, solution queuerepo.Solution) error {
	_, err := q.queue.ExecContext(ctx, "INSERT INTO queue (taskid, userid, status, created_at, solution) VALUES($1, $2, $3, $4,$5)", solution.TaskID, solution.UserID, solution.Status, solution.CreatedAt, solution.Solution)
	if err != nil {
		return err
	}

	return nil
}
