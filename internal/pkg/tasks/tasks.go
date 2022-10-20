package tasks

import (
	"context"
	"errors"
)

var (
	ErrNoTask = errors.New("no task with given params found")
	ErrNoUser = errors.New("no user with given name found")
)

type Task struct {
	ID          uint32
	Name        string
	Description string
}

type User struct {
	ID       uint64
	Name     string
	Password string
	IsAdmin  bool
}

type Tasker interface {
	GetAllTasks(context.Context) ([]*Task, error)
	CreateTask(context.Context, *Task) error
	GetTaskByID(context.Context, uint32) (*Task, error)

	GetUserByName(context.Context, string) (*User, error)
	GetUserById(context.Context, uint64) (*User, error)
	CreateUser(context.Context, *User) (*User, error)

	Close() error
}
