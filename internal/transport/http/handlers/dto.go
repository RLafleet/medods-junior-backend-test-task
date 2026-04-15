package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/template"
	templateusecase "example.com/taskservice/internal/usecase/template"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
}

type taskDTO struct {
	ID           int64             `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Status       taskdomain.Status `json:"status"`
	TemplateID   *int64            `json:"template_id,omitempty"`
	ScheduledFor *string           `json:"scheduled_for,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		TemplateID:  task.TemplateID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}

	if task.ScheduledFor != nil {
		s := task.ScheduledFor.Format("2006-01-02")
		dto.ScheduledFor = &s
	}

	return dto
}

type templateMutationDTO struct {
	Title          string                         `json:"title"`
	Description    string                         `json:"description"`
	RecurrenceType templatedomain.RecurrenceType  `json:"recurrence_type"`
	StartDate      string                         `json:"start_date"`
	Timezone       string                         `json:"timezone"`
	EveryNDays     *int                           `json:"every_n_days,omitempty"`
	DayOfMonth     *int                           `json:"day_of_month,omitempty"`
	SpecificDates  []string                       `json:"specific_dates,omitempty"`
	MonthDayParity *templatedomain.MonthDayParity `json:"month_day_parity,omitempty"`
}

func (d templateMutationDTO) toCreateInput() templateusecase.CreateInput {
	return templateusecase.CreateInput{
		Title:          d.Title,
		Description:    d.Description,
		RecurrenceType: d.RecurrenceType,
		StartDate:      parseDateString(d.StartDate),
		Timezone:       d.Timezone,
		EveryNDays:     d.EveryNDays,
		DayOfMonth:     d.DayOfMonth,
		SpecificDates:  parseDateStrings(d.SpecificDates),
		MonthDayParity: d.MonthDayParity,
	}
}

func (d templateMutationDTO) toUpdateInput() templateusecase.UpdateInput {
	return templateusecase.UpdateInput{
		Title:          d.Title,
		Description:    d.Description,
		RecurrenceType: d.RecurrenceType,
		StartDate:      parseDateString(d.StartDate),
		Timezone:       d.Timezone,
		EveryNDays:     d.EveryNDays,
		DayOfMonth:     d.DayOfMonth,
		SpecificDates:  parseDateStrings(d.SpecificDates),
		MonthDayParity: d.MonthDayParity,
	}
}

type templateDTO struct {
	ID             int64                          `json:"id"`
	Title          string                         `json:"title"`
	Description    string                         `json:"description"`
	IsActive       bool                           `json:"is_active"`
	RecurrenceType templatedomain.RecurrenceType  `json:"recurrence_type"`
	StartDate      string                         `json:"start_date"`
	Timezone       string                         `json:"timezone"`
	EveryNDays     *int                           `json:"every_n_days,omitempty"`
	DayOfMonth     *int                           `json:"day_of_month,omitempty"`
	SpecificDates  []string                       `json:"specific_dates,omitempty"`
	MonthDayParity *templatedomain.MonthDayParity `json:"month_day_parity,omitempty"`
	CreatedAt      time.Time                      `json:"created_at"`
	UpdatedAt      time.Time                      `json:"updated_at"`
}

func newTemplateDTO(t *templatedomain.Template) templateDTO {
	dto := templateDTO{
		ID:             t.ID,
		Title:          t.Title,
		Description:    t.Description,
		IsActive:       t.IsActive,
		RecurrenceType: t.RecurrenceType,
		StartDate:      t.StartDate.Format("2006-01-02"),
		Timezone:       t.Timezone,
		EveryNDays:     t.EveryNDays,
		DayOfMonth:     t.DayOfMonth,
		MonthDayParity: t.MonthDayParity,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}

	if len(t.SpecificDates) > 0 {
		dto.SpecificDates = make([]string, len(t.SpecificDates))
		for i, d := range t.SpecificDates {
			dto.SpecificDates[i] = d.Format("2006-01-02")
		}
	}

	return dto
}

func parseDateString(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseDateStrings(ss []string) []time.Time {
	if len(ss) == 0 {
		return nil
	}

	result := make([]time.Time, 0, len(ss))
	for _, s := range ss {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			continue
		}
		result = append(result, t)
	}

	return result
}
