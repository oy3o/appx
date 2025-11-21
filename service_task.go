package appx

import (
	"context"

	"github.com/oy3o/task"
)

type TaskService struct {
	runner *task.Runner
}

func NewTaskService(runner *task.Runner) Service {
	return &TaskService{runner: runner}
}

func (t *TaskService) Name() string { return "background-tasks" }

func (t *TaskService) Start(ctx context.Context) error {
	return t.runner.Start(ctx)
}

func (t *TaskService) Stop(ctx context.Context) error {
	return t.runner.Stop(ctx)
}
