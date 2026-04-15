package template

import (
	"context"
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/template"
)

func intPtr(n int) *int { return &n }

func parityPtr(p templatedomain.MonthDayParity) *templatedomain.MonthDayParity { return &p }

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

type mockTemplateRepo struct {
	templates []templatedomain.Template
}

func (m *mockTemplateRepo) Create(_ context.Context, t *templatedomain.Template) (*templatedomain.Template, error) {
	t.ID = int64(len(m.templates) + 1)
	m.templates = append(m.templates, *t)
	return t, nil
}

func (m *mockTemplateRepo) GetByID(_ context.Context, id int64) (*templatedomain.Template, error) {
	for _, t := range m.templates {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, templatedomain.ErrNotFound
}

func (m *mockTemplateRepo) Update(_ context.Context, t *templatedomain.Template) (*templatedomain.Template, error) {
	for i, existing := range m.templates {
		if existing.ID == t.ID {
			m.templates[i] = *t
			return t, nil
		}
	}
	return nil, templatedomain.ErrNotFound
}

func (m *mockTemplateRepo) Deactivate(_ context.Context, id int64) error {
	for i, t := range m.templates {
		if t.ID == id {
			m.templates[i].IsActive = false
			return nil
		}
	}
	return templatedomain.ErrNotFound
}

func (m *mockTemplateRepo) List(_ context.Context) ([]templatedomain.Template, error) {
	return m.templates, nil
}

func (m *mockTemplateRepo) ListActive(_ context.Context) ([]templatedomain.Template, error) {
	var result []templatedomain.Template
	for _, t := range m.templates {
		if t.IsActive {
			result = append(result, t)
		}
	}
	return result, nil
}

type mockTaskRepo struct {
	created []taskdomain.Task
}

func (m *mockTaskRepo) CreateFromTemplate(_ context.Context, task *taskdomain.Task) error {
	m.created = append(m.created, *task)
	return nil
}

func newTestService() (*Service, *mockTemplateRepo, *mockTaskRepo) {
	tRepo := &mockTemplateRepo{}
	taskRepo := &mockTaskRepo{}
	svc := NewService(tRepo, taskRepo)
	return svc, tRepo, taskRepo
}

func TestGenerateDates_Daily(t *testing.T) {
	tmpl := templatedomain.Template{
		RecurrenceType: templatedomain.RecurrenceDaily,
		StartDate:      date(2024, 1, 1),
		EveryNDays:     intPtr(3),
	}

	from := date(2024, 1, 1)
	until := date(2024, 1, 10)
	got := generateDates(tmpl, from, until)

	want := []time.Time{
		date(2024, 1, 1),
		date(2024, 1, 4),
		date(2024, 1, 7),
		date(2024, 1, 10),
	}

	assertDates(t, want, got)
}

func TestGenerateDates_DailyAlignedToStartDate(t *testing.T) {
	tmpl := templatedomain.Template{
		RecurrenceType: templatedomain.RecurrenceDaily,
		StartDate:      date(2024, 1, 1),
		EveryNDays:     intPtr(5),
	}

	from := date(2024, 1, 8)
	until := date(2024, 1, 20)
	got := generateDates(tmpl, from, until)

	want := []time.Time{
		date(2024, 1, 11),
		date(2024, 1, 16),
	}

	assertDates(t, want, got)
}

func TestGenerateDates_Monthly_Normal(t *testing.T) {
	tmpl := templatedomain.Template{
		RecurrenceType: templatedomain.RecurrenceMonthly,
		StartDate:      date(2024, 1, 1),
		DayOfMonth:     intPtr(15),
	}

	from := date(2024, 1, 1)
	until := date(2024, 3, 31)
	got := generateDates(tmpl, from, until)

	want := []time.Time{
		date(2024, 1, 15),
		date(2024, 2, 15),
		date(2024, 3, 15),
	}

	assertDates(t, want, got)
}

func TestGenerateDates_Monthly_Day30InFebruary(t *testing.T) {
	tmpl := templatedomain.Template{
		RecurrenceType: templatedomain.RecurrenceMonthly,
		StartDate:      date(2024, 1, 1),
		DayOfMonth:     intPtr(30),
	}

	from := date(2024, 1, 1)
	until := date(2024, 4, 30)
	got := generateDates(tmpl, from, until)

	want := []time.Time{
		date(2024, 1, 30),
		date(2024, 3, 30),
		date(2024, 4, 30),
	}

	assertDates(t, want, got)
}

func TestGenerateDates_Monthly_Day30InOctober(t *testing.T) {
	tmpl := templatedomain.Template{
		RecurrenceType: templatedomain.RecurrenceMonthly,
		StartDate:      date(2024, 1, 1),
		DayOfMonth:     intPtr(30),
	}

	from := date(2024, 10, 1)
	until := date(2024, 10, 31)
	got := generateDates(tmpl, from, until)

	want := []time.Time{
		date(2024, 10, 30),
	}

	assertDates(t, want, got)
}

func TestGenerateDates_SpecificDates_Filtering(t *testing.T) {
	tmpl := templatedomain.Template{
		RecurrenceType: templatedomain.RecurrenceSpecificDates,
		SpecificDates: []time.Time{
			date(2024, 1, 5),
			date(2024, 1, 20),
			date(2024, 2, 10),
			date(2024, 3, 1),
		},
	}

	from := date(2024, 1, 10)
	until := date(2024, 2, 28)
	got := generateDates(tmpl, from, until)

	want := []time.Time{
		date(2024, 1, 20),
		date(2024, 2, 10),
	}

	assertDates(t, want, got)
}

func TestGenerateDates_EvenOdd_Even(t *testing.T) {
	tmpl := templatedomain.Template{
		RecurrenceType: templatedomain.RecurrenceEvenOdd,
		StartDate:      date(2024, 1, 1),
		MonthDayParity: parityPtr(templatedomain.ParityEven),
	}

	from := date(2024, 1, 1)
	until := date(2024, 1, 6)
	got := generateDates(tmpl, from, until)

	want := []time.Time{
		date(2024, 1, 2),
		date(2024, 1, 4),
		date(2024, 1, 6),
	}

	assertDates(t, want, got)
}

func TestGenerateDates_EvenOdd_Odd(t *testing.T) {
	tmpl := templatedomain.Template{
		RecurrenceType: templatedomain.RecurrenceEvenOdd,
		StartDate:      date(2024, 1, 1),
		MonthDayParity: parityPtr(templatedomain.ParityOdd),
	}

	from := date(2024, 1, 1)
	until := date(2024, 1, 5)
	got := generateDates(tmpl, from, until)

	want := []time.Time{
		date(2024, 1, 1),
		date(2024, 1, 3),
		date(2024, 1, 5),
	}

	assertDates(t, want, got)
}

func TestValidation_EveryNDaysZero(t *testing.T) {
	svc, _, _ := newTestService()
	n := 0
	_, err := svc.Create(context.Background(), CreateInput{
		Title:          "t",
		RecurrenceType: templatedomain.RecurrenceDaily,
		StartDate:      date(2024, 1, 1),
		Timezone:       "UTC",
		EveryNDays:     &n,
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for every_n_days=0, got %v", err)
	}
}

func TestValidation_EveryNDaysNegative(t *testing.T) {
	svc, _, _ := newTestService()
	n := -1
	_, err := svc.Create(context.Background(), CreateInput{
		Title:          "t",
		RecurrenceType: templatedomain.RecurrenceDaily,
		StartDate:      date(2024, 1, 1),
		Timezone:       "UTC",
		EveryNDays:     &n,
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for every_n_days=-1, got %v", err)
	}
}

func TestValidation_DayOfMonthZero(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Create(context.Background(), CreateInput{
		Title:          "t",
		RecurrenceType: templatedomain.RecurrenceMonthly,
		StartDate:      date(2024, 1, 1),
		Timezone:       "UTC",
		DayOfMonth:     intPtr(0),
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for day_of_month=0, got %v", err)
	}
}

func TestValidation_DayOfMonth31(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Create(context.Background(), CreateInput{
		Title:          "t",
		RecurrenceType: templatedomain.RecurrenceMonthly,
		StartDate:      date(2024, 1, 1),
		Timezone:       "UTC",
		DayOfMonth:     intPtr(31),
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for day_of_month=31, got %v", err)
	}
}

func TestValidation_SpecificDatesEmpty(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Create(context.Background(), CreateInput{
		Title:          "t",
		RecurrenceType: templatedomain.RecurrenceSpecificDates,
		StartDate:      date(2024, 1, 1),
		Timezone:       "UTC",
		SpecificDates:  []time.Time{},
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty specific_dates, got %v", err)
	}
}

func TestValidation_SpecificDatesDuplicate(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Create(context.Background(), CreateInput{
		Title:          "t",
		RecurrenceType: templatedomain.RecurrenceSpecificDates,
		StartDate:      date(2024, 1, 1),
		Timezone:       "UTC",
		SpecificDates:  []time.Time{date(2024, 1, 5), date(2024, 1, 5)},
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for duplicate specific_dates, got %v", err)
	}
}

func TestValidation_InvalidTimezone(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Create(context.Background(), CreateInput{
		Title:          "t",
		RecurrenceType: templatedomain.RecurrenceDaily,
		StartDate:      date(2024, 1, 1),
		Timezone:       "Not/ATimezone",
		EveryNDays:     intPtr(1),
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for invalid timezone, got %v", err)
	}
}

func TestGenerateUpTo_Idempotent(t *testing.T) {
	tRepo := &mockTemplateRepo{}
	callCount := 0
	idempotentTaskRepo := &idempotentMockTaskRepo{&callCount}
	svc := NewService(tRepo, idempotentTaskRepo)
	svc.now = func() time.Time { return date(2024, 1, 1) }

	n := 1
	tRepo.templates = []templatedomain.Template{
		{
			ID:             1,
			Title:          "Daily",
			IsActive:       true,
			RecurrenceType: templatedomain.RecurrenceDaily,
			StartDate:      date(2024, 1, 1),
			Timezone:       "UTC",
			EveryNDays:     &n,
		},
	}

	until := date(2024, 1, 3)

	if err := svc.GenerateUpTo(context.Background(), until); err != nil {
		t.Fatal(err)
	}

	firstCount := callCount

	if err := svc.GenerateUpTo(context.Background(), until); err != nil {
		t.Fatal(err)
	}

	if callCount != firstCount*2 {
		t.Fatalf("expected %d total calls on second run (idempotent), got %d", firstCount*2, callCount)
	}
}

func TestGenerateUpTo_TemplateUpdateAffectsOnlyFuture(t *testing.T) {
	svc, tRepo, taskRepo := newTestService()
	svc.now = func() time.Time { return date(2024, 1, 5) }

	n := 1
	tRepo.templates = []templatedomain.Template{
		{
			ID:             1,
			Title:          "Daily",
			IsActive:       true,
			RecurrenceType: templatedomain.RecurrenceDaily,
			StartDate:      date(2024, 1, 1),
			Timezone:       "UTC",
			EveryNDays:     &n,
		},
	}

	until := date(2024, 1, 7)
	if err := svc.GenerateUpTo(context.Background(), until); err != nil {
		t.Fatal(err)
	}

	firstBatch := len(taskRepo.created)
	if firstBatch == 0 {
		t.Fatal("expected tasks to be created")
	}

	tRepo.templates[0].Title = "Updated Title"

	if err := svc.GenerateUpTo(context.Background(), until); err != nil {
		t.Fatal(err)
	}

	for _, task := range taskRepo.created[:firstBatch] {
		if task.Title != "Daily" {
			t.Errorf("existing task title was retroactively changed to %q", task.Title)
		}
	}
}

type idempotentMockTaskRepo struct {
	callCount *int
}

func (m *idempotentMockTaskRepo) CreateFromTemplate(_ context.Context, _ *taskdomain.Task) error {
	*m.callCount++
	return nil
}

func assertDates(t *testing.T, want, got []time.Time) {
	t.Helper()

	if len(want) != len(got) {
		t.Fatalf("expected %d dates, got %d\nwant: %v\ngot:  %v", len(want), len(got), want, got)
	}

	for i := range want {
		if !want[i].Equal(got[i]) {
			t.Errorf("date[%d]: want %s, got %s", i, want[i].Format("2006-01-02"), got[i].Format("2006-01-02"))
		}
	}
}
