package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	templatedomain "example.com/taskservice/internal/domain/template"
)

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

func (r *TemplateRepository) Create(ctx context.Context, t *templatedomain.Template) (*templatedomain.Template, error) {
	const query = `
		INSERT INTO task_templates (
			title, description, is_active, recurrence_type, start_date, timezone,
			every_n_days, day_of_month, specific_dates, month_day_parity,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, title, description, is_active, recurrence_type, start_date, timezone,
			every_n_days, day_of_month, specific_dates, month_day_parity,
			created_at, updated_at
	`

	specificDates := timesToPgDates(t.SpecificDates)
	row := r.pool.QueryRow(ctx, query,
		t.Title, t.Description, t.IsActive, string(t.RecurrenceType),
		pgDateFromTime(t.StartDate), t.Timezone,
		t.EveryNDays, t.DayOfMonth, specificDates, monthDayParityPtr(t.MonthDayParity),
		t.CreatedAt, t.UpdatedAt,
	)

	return scanTemplate(row)
}

func (r *TemplateRepository) GetByID(ctx context.Context, id int64) (*templatedomain.Template, error) {
	const query = `
		SELECT id, title, description, is_active, recurrence_type, start_date, timezone,
			every_n_days, day_of_month, specific_dates, month_day_parity,
			created_at, updated_at
		FROM task_templates
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, templatedomain.ErrNotFound
		}
		return nil, err
	}

	return found, nil
}

func (r *TemplateRepository) Update(ctx context.Context, t *templatedomain.Template) (*templatedomain.Template, error) {
	const query = `
		UPDATE task_templates
		SET title            = $1,
		    description      = $2,
		    is_active        = $3,
		    recurrence_type  = $4,
		    start_date       = $5,
		    timezone         = $6,
		    every_n_days     = $7,
		    day_of_month     = $8,
		    specific_dates   = $9,
		    month_day_parity = $10,
		    updated_at       = $11
		WHERE id = $12
		RETURNING id, title, description, is_active, recurrence_type, start_date, timezone,
			every_n_days, day_of_month, specific_dates, month_day_parity,
			created_at, updated_at
	`

	specificDates := timesToPgDates(t.SpecificDates)
	row := r.pool.QueryRow(ctx, query,
		t.Title, t.Description, t.IsActive, string(t.RecurrenceType),
		pgDateFromTime(t.StartDate), t.Timezone,
		t.EveryNDays, t.DayOfMonth, specificDates, monthDayParityPtr(t.MonthDayParity),
		t.UpdatedAt, t.ID,
	)

	updated, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, templatedomain.ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

func (r *TemplateRepository) Deactivate(ctx context.Context, id int64) error {
	const query = `UPDATE task_templates SET is_active = FALSE, updated_at = NOW() WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return templatedomain.ErrNotFound
	}

	return nil
}

func (r *TemplateRepository) List(ctx context.Context) ([]templatedomain.Template, error) {
	const query = `
		SELECT id, title, description, is_active, recurrence_type, start_date, timezone,
			every_n_days, day_of_month, specific_dates, month_day_parity,
			created_at, updated_at
		FROM task_templates
		ORDER BY id DESC
	`

	return r.queryTemplates(ctx, query)
}

func (r *TemplateRepository) ListActive(ctx context.Context) ([]templatedomain.Template, error) {
	const query = `
		SELECT id, title, description, is_active, recurrence_type, start_date, timezone,
			every_n_days, day_of_month, specific_dates, month_day_parity,
			created_at, updated_at
		FROM task_templates
		WHERE is_active = TRUE
		ORDER BY id DESC
	`

	return r.queryTemplates(ctx, query)
}

func (r *TemplateRepository) queryTemplates(ctx context.Context, query string) ([]templatedomain.Template, error) {
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]templatedomain.Template, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return templates, nil
}

type templateScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(scanner templateScanner) (*templatedomain.Template, error) {
	var (
		t              templatedomain.Template
		recurrenceType string
		startDate      pgtype.Date
		everyNDays     *int
		dayOfMonth     *int
		specificDates  pgtype.Array[pgtype.Date]
		parity         *string
	)

	if err := scanner.Scan(
		&t.ID,
		&t.Title,
		&t.Description,
		&t.IsActive,
		&recurrenceType,
		&startDate,
		&t.Timezone,
		&everyNDays,
		&dayOfMonth,
		&specificDates,
		&parity,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return nil, err
	}

	t.RecurrenceType = templatedomain.RecurrenceType(recurrenceType)
	if startDate.Valid {
		t.StartDate = startDate.Time
	}
	t.EveryNDays = everyNDays
	t.DayOfMonth = dayOfMonth

	if specificDates.Valid {
		for _, d := range specificDates.Elements {
			if d.Valid {
				t.SpecificDates = append(t.SpecificDates, d.Time)
			}
		}
	}
	if t.SpecificDates == nil {
		t.SpecificDates = []time.Time{}
	}

	if parity != nil {
		p := templatedomain.MonthDayParity(*parity)
		t.MonthDayParity = &p
	}

	return &t, nil
}

func pgDateFromTime(t time.Time) pgtype.Date {
	return pgtype.Date{
		Time:  t,
		Valid: true,
	}
}

func timesToPgDates(times []time.Time) pgtype.Array[pgtype.Date] {
	if len(times) == 0 {
		return pgtype.Array[pgtype.Date]{Valid: false}
	}

	elements := make([]pgtype.Date, len(times))
	for i, t := range times {
		elements[i] = pgtype.Date{Time: t, Valid: true}
	}

	return pgtype.Array[pgtype.Date]{
		Elements: elements,
		Dims:     []pgtype.ArrayDimension{{Length: int32(len(elements)), LowerBound: 1}},
		Valid:    true,
	}
}

func monthDayParityPtr(p *templatedomain.MonthDayParity) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}
