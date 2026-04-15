package template

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/template"
)

type TemplateRepository interface {
	Create(ctx context.Context, t *templatedomain.Template) (*templatedomain.Template, error)
	GetByID(ctx context.Context, id int64) (*templatedomain.Template, error)
	Update(ctx context.Context, t *templatedomain.Template) (*templatedomain.Template, error)
	Deactivate(ctx context.Context, id int64) error
	List(ctx context.Context) ([]templatedomain.Template, error)
	ListActive(ctx context.Context) ([]templatedomain.Template, error)
}

type TaskRepository interface {
	CreateFromTemplate(ctx context.Context, task *taskdomain.Task) error
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*templatedomain.Template, error)
	GetByID(ctx context.Context, id int64) (*templatedomain.Template, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*templatedomain.Template, error)
	Deactivate(ctx context.Context, id int64) error
	List(ctx context.Context) ([]templatedomain.Template, error)
	GenerateUpTo(ctx context.Context, until time.Time) error
}

type CreateInput struct {
	Title          string
	Description    string
	RecurrenceType templatedomain.RecurrenceType
	StartDate      time.Time
	Timezone       string
	EveryNDays     *int
	DayOfMonth     *int
	SpecificDates  []time.Time
	MonthDayParity *templatedomain.MonthDayParity
}

type UpdateInput struct {
	Title          string
	Description    string
	RecurrenceType templatedomain.RecurrenceType
	StartDate      time.Time
	Timezone       string
	EveryNDays     *int
	DayOfMonth     *int
	SpecificDates  []time.Time
	MonthDayParity *templatedomain.MonthDayParity
}
