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
	_, err := q.queue.ExecContext(ctx, "INSERT INTO queue ()")
	if err != nil {
		return err
	}
	return nil

}
