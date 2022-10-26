package tasks

import (
	"context"
	"database/sql"
	"fmt"

	tasksrepo "github.com/kateshostak/grader/internal/pkg/tasks"
)

type TasksRepo struct {
	tasks *sql.DB
}

func NewTasker(db *sql.DB) *TasksRepo {
	return &TasksRepo{
		tasks: db,
	}
}

func (t TasksRepo) CreateTask(ctx context.Context, task *tasksrepo.Task) error {
	if _, err := t.tasks.ExecContext(ctx, "INSERT INTO tasks (name, description) VALUES($1, $2)", task.Name, task.Description); err != nil {
		return err
	}

	return nil
}

func (t TasksRepo) GetAllTasks(ctx context.Context) ([]*tasksrepo.Task, error) {
	curr, err := t.tasks.QueryContext(ctx, "SELECT id, name, description FROM tasks")
	if err != nil {
		return nil, err
	}

	res := make([]*tasksrepo.Task, 0)
	for curr.Next() {
		var task tasksrepo.Task
		if err := curr.Scan(&task.ID, &task.Name, &task.Description); err != nil {
			return nil, err
		}
		res = append(res, &task)
	}
	return res, nil
}

func (t TasksRepo) GetTaskByID(ctx context.Context, id uint32) (*tasksrepo.Task, error) {
	var task tasksrepo.Task
	if err := t.tasks.QueryRowContext(ctx, "SELECT id, name, description FROM tasks where id=$1", id).Scan(&task.ID, &task.Name, &task.Description); err != nil {
		if err == sql.ErrNoRows {
			return nil, tasksrepo.ErrNoTask
		}
		return nil, err
	}
	return &task, nil
}

func (t TasksRepo) CreateUser(ctx context.Context, user *tasksrepo.User) (*tasksrepo.User, error) {
	var id uint64

	if err := t.tasks.QueryRowContext(ctx, "INSERT INTO users (name, password) VALUES($1, $2) returning id", user.Name, user.Password).Scan(&id); err != nil {
		return nil, err
	}

	user.ID = id
	return user, nil
}

func (t TasksRepo) GetUserByName(ctx context.Context, name string) (*tasksrepo.User, error) {
	var user tasksrepo.User

	if err := t.tasks.QueryRowContext(ctx, "SELECT id, name, password FROM users WHERE name=$1", name).Scan(&user.ID, &user.Name, &user.Password); err != nil {
		if err == sql.ErrNoRows {
			return nil, tasksrepo.ErrNoUser
		}
		return nil, err
	}

	return &user, nil
}

func (t TasksRepo) GetUserById(ctx context.Context, id uint64) (*tasksrepo.User, error) {
	var user tasksrepo.User

	if err := t.tasks.QueryRowContext(ctx, "SELECT id, name, password, isadmin FROM users WHERE id=$1", id).Scan(&user.ID, &user.Name, &user.Password, &user.IsAdmin); err != nil {
		if err == sql.ErrNoRows {
			return nil, tasksrepo.ErrNoUser
		}
		return nil, err
	}

	return &user, nil
}

func (t TasksRepo) Close() error {
	if err := t.tasks.Close(); err != nil {
		return fmt.Errorf("cant close tasksDB: %v", err)
	}
	return nil
}
