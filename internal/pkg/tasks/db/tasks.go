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

func (t *TasksRepo) CreateTask(ctx context.Context, task *tasksrepo.Task) error {
	if _, err := t.tasks.ExecContext(ctx, "INSERT INTO tasks (name, description) VALUES($1, $2)", task.Name, task.Description); err != nil {
		return err
	}

	return nil
}

func (t *TasksRepo) GetAllTasks(ctx context.Context) ([]*tasksrepo.Task, error) {
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

func (t *TasksRepo) GetTaskByID(ctx context.Context, id uint32) (*tasksrepo.Task, error) {
	var task tasksrepo.Task
	if err := t.tasks.QueryRowContext(ctx, "SELECT id, name, description FROM tasks where id=$1", id).Scan(&task.ID, &task.Name, &task.Description); err != nil {
		return nil, err
	}
	return &task, nil
}

func (t *TasksRepo) Close() error {
	if err := t.tasks.Close(); err != nil {
		return fmt.Errorf("cant close tasksDB: %v", err)
	}
	return nil
}
