package queue

import (
	"context"
	"time"
)

type Solution struct {
	TaskID    uint32
	UserID    uint64
	CreatedAt time.Time
	Solution  []byte
	Status    string
}

type Queuer interface {
	Add(context.Context, Solution) error
}
