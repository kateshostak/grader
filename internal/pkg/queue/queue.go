package queue

import (
	"context"
)

type Solution struct {
	TaskID   uint32
	UserID   uint64
	Solution []byte
	Status   string
}

type Queuer interface {
	Add(context.Context, Solution) error
}
