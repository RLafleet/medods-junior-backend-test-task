package template

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/template"
)

type Service struct {
	templateRepo TemplateRepository
	taskRepo     TaskRepository
	now          func() time.Time
}

func NewService(templateRepo TemplateRepository, taskRepo TaskRepository) *Service {
	return &Service{
		templateRepo: templateRepo,
		taskRepo:     taskRepo,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*templatedomain.Template, error) {
	normalized, err := validateInput(input.Title, input.Description, input.RecurrenceType,
		input.StartDate, input.Timezone, input.EveryNDays, input.DayOfMonth,
		input.SpecificDates, input.MonthDayParity)
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &templatedomain.Template{
		Title:          normalized.title,
		Description:    normalized.description,
		IsActive:       true,
		RecurrenceType: normalized.recurrenceType,
		StartDate:      normalized.startDate,
		Timezone:       normalized.timezone,
		EveryNDays:     normalized.everyNDays,
		DayOfMonth:     normalized.dayOfMonth,
		SpecificDates:  normalized.specificDates,
		MonthDayParity: normalized.monthDayParity,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return s.templateRepo.Create(ctx, model)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*templatedomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.templateRepo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*templatedomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	existing, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	normalized, err := validateInput(input.Title, input.Description, input.RecurrenceType,
		input.StartDate, input.Timezone, input.EveryNDays, input.DayOfMonth,
		input.SpecificDates, input.MonthDayParity)
	if err != nil {
		return nil, err
	}

	model := &templatedomain.Template{
		ID:             id,
		Title:          normalized.title,
		Description:    normalized.description,
		IsActive:       existing.IsActive,
		RecurrenceType: normalized.recurrenceType,
		StartDate:      normalized.startDate,
		Timezone:       normalized.timezone,
		EveryNDays:     normalized.everyNDays,
		DayOfMonth:     normalized.dayOfMonth,
		SpecificDates:  normalized.specificDates,
		MonthDayParity: normalized.monthDayParity,
		UpdatedAt:      s.now(),
	}

	updated, err := s.templateRepo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Deactivate(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	if err := s.templateRepo.Deactivate(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *Service) List(ctx context.Context) ([]templatedomain.Template, error) {
	return s.templateRepo.List(ctx)
}

func (s *Service) GenerateUpTo(ctx context.Context, until time.Time) error {
	templates, err := s.templateRepo.ListActive(ctx)
	if err != nil {
		return err
	}

	for _, t := range templates {
		loc, err := loadLocationOrUTC(t.Timezone)
		if err != nil {
			return fmt.Errorf("invalid template timezone %q for template_id=%d: %w", t.Timezone, t.ID, err)
		}

		today := truncateToDateInLocation(s.now(), loc)
		from := civilDateInLocation(t.StartDate, loc)
		if today.After(from) {
			from = today
		}

		untilLocal := truncateToDateInLocation(until, loc)
		dates := generateDatesInLocation(t, from, untilLocal, loc)

		for _, d := range dates {
			scheduledFor := dateAsUTCMidnight(d)
			task := &taskdomain.Task{
				Title:        t.Title,
				Description:  t.Description,
				Status:       taskdomain.StatusNew,
				TemplateID:   &t.ID,
				ScheduledFor: &scheduledFor,
				CreatedAt:    s.now(),
				UpdatedAt:    s.now(),
			}

			if err := s.taskRepo.CreateFromTemplate(ctx, task); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateDates(t templatedomain.Template, from, until time.Time) []time.Time {
	return generateDatesInLocation(t, from, until, from.Location())
}

func generateDatesInLocation(t templatedomain.Template, from, until time.Time, loc *time.Location) []time.Time {
	if loc == nil {
		loc = time.UTC
	}

	var dates []time.Time

	switch t.RecurrenceType {
	case templatedomain.RecurrenceDaily:
		if t.EveryNDays == nil || *t.EveryNDays <= 0 {
			return nil
		}

		n := *t.EveryNDays
		start := civilDateInLocation(t.StartDate, loc)
		rangeStart := from
		if rangeStart.Before(start) {
			rangeStart = start
		}

		diff := daysBetweenDates(start, rangeStart)
		offset := diff % n
		firstInRange := rangeStart
		if offset != 0 {
			firstInRange = rangeStart.AddDate(0, 0, n-offset)
		}

		for d := firstInRange; !d.After(until); d = d.AddDate(0, 0, n) {
			dates = append(dates, d)
		}

	case templatedomain.RecurrenceMonthly:
		if t.DayOfMonth == nil {
			return nil
		}

		day := *t.DayOfMonth

		for year, month := from.Year(), from.Month(); ; {
			candidate := time.Date(year, month, 1, 0, 0, 0, 0, loc)
			if candidate.After(until) {
				break
			}

			daysInMonth := daysIn(year, month, loc)
			if day <= daysInMonth {
				d := time.Date(year, month, day, 0, 0, 0, 0, loc)
				if !d.Before(from) && !d.After(until) {
					dates = append(dates, d)
				}
			}

			month++
			if month > 12 {
				month = 1
				year++
			}
		}

	case templatedomain.RecurrenceSpecificDates:
		for _, sd := range t.SpecificDates {
			d := civilDateInLocation(sd, loc)
			if !d.Before(from) && !d.After(until) {
				dates = append(dates, d)
			}
		}

	case templatedomain.RecurrenceEvenOdd:
		if t.MonthDayParity == nil {
			return nil
		}

		wantEven := *t.MonthDayParity == templatedomain.ParityEven

		for d := from; !d.After(until); d = d.AddDate(0, 0, 1) {
			dayNum := d.Day()
			isEven := dayNum%2 == 0
			if isEven == wantEven {
				dates = append(dates, d)
			}
		}
	}

	return dates
}

type validatedInput struct {
	title          string
	description    string
	recurrenceType templatedomain.RecurrenceType
	startDate      time.Time
	timezone       string
	everyNDays     *int
	dayOfMonth     *int
	specificDates  []time.Time
	monthDayParity *templatedomain.MonthDayParity
}

func validateInput(
	title, description string,
	recurrenceType templatedomain.RecurrenceType,
	startDate time.Time,
	timezone string,
	everyNDays, dayOfMonth *int,
	specificDates []time.Time,
	monthDayParity *templatedomain.MonthDayParity,
) (validatedInput, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	if title == "" {
		return validatedInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !recurrenceType.Valid() {
		return validatedInput{}, fmt.Errorf("%w: invalid recurrence_type", ErrInvalidInput)
	}

	if timezone == "" {
		timezone = "UTC"
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return validatedInput{}, fmt.Errorf("%w: invalid timezone: %s", ErrInvalidInput, timezone)
	}

	if startDate.IsZero() {
		return validatedInput{}, fmt.Errorf("%w: start_date is required", ErrInvalidInput)
	}

	switch recurrenceType {
	case templatedomain.RecurrenceDaily:
		if everyNDays == nil || *everyNDays <= 0 {
			return validatedInput{}, fmt.Errorf("%w: every_n_days must be positive for daily recurrence", ErrInvalidInput)
		}

	case templatedomain.RecurrenceMonthly:
		if dayOfMonth == nil || *dayOfMonth < 1 || *dayOfMonth > 30 {
			return validatedInput{}, fmt.Errorf("%w: day_of_month must be between 1 and 30 for monthly recurrence", ErrInvalidInput)
		}

	case templatedomain.RecurrenceSpecificDates:
		if len(specificDates) == 0 {
			return validatedInput{}, fmt.Errorf("%w: specific_dates must not be empty", ErrInvalidInput)
		}

		seen := make(map[string]struct{}, len(specificDates))
		deduped := make([]time.Time, 0, len(specificDates))

		for _, d := range specificDates {
			key := d.Format("2006-01-02")
			if _, exists := seen[key]; exists {
				return validatedInput{}, fmt.Errorf("%w: specific_dates contains duplicate date %s", ErrInvalidInput, key)
			}
			seen[key] = struct{}{}
			deduped = append(deduped, truncateToDate(d))
		}

		specificDates = deduped

	case templatedomain.RecurrenceEvenOdd:
		if monthDayParity == nil || !monthDayParity.Valid() {
			return validatedInput{}, fmt.Errorf("%w: month_day_parity must be 'even' or 'odd' for even_odd recurrence", ErrInvalidInput)
		}
	}

	return validatedInput{
		title:          title,
		description:    description,
		recurrenceType: recurrenceType,
		startDate:      truncateToDate(startDate),
		timezone:       timezone,
		everyNDays:     everyNDays,
		dayOfMonth:     dayOfMonth,
		specificDates:  specificDates,
		monthDayParity: monthDayParity,
	}, nil
}

func truncateToDate(t time.Time) time.Time {
	return dateAsUTCMidnight(t)
}

func truncateToDateInLocation(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}

	localTime := t.In(loc)
	return time.Date(localTime.Year(), localTime.Month(), localTime.Day(), 0, 0, 0, 0, loc)
}

func civilDateInLocation(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}

	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

func dateAsUTCMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func daysIn(year int, month time.Month, loc *time.Location) int {
	if loc == nil {
		loc = time.UTC
	}
	return time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
}

func daysBetweenDates(from, to time.Time) int {
	return dateOrdinal(to) - dateOrdinal(from)
}

func dateOrdinal(t time.Time) int {
	year, month, day := t.Date()
	return int(time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

func loadLocationOrUTC(timezone string) (*time.Location, error) {
	if timezone == "" {
		timezone = "UTC"
	}

	return time.LoadLocation(timezone)
}
