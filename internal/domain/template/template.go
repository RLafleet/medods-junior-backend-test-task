package template

import "time"

type RecurrenceType string

const (
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthly       RecurrenceType = "monthly"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceEvenOdd       RecurrenceType = "even_odd"
)

func (r RecurrenceType) Valid() bool {
	switch r {
	case RecurrenceDaily, RecurrenceMonthly, RecurrenceSpecificDates, RecurrenceEvenOdd:
		return true
	default:
		return false
	}
}

type MonthDayParity string

const (
	ParityEven MonthDayParity = "even"
	ParityOdd  MonthDayParity = "odd"
)

func (p MonthDayParity) Valid() bool {
	return p == ParityEven || p == ParityOdd
}

type Template struct {
	ID             int64
	Title          string
	Description    string
	IsActive       bool
	RecurrenceType RecurrenceType
	StartDate      time.Time
	Timezone       string
	EveryNDays     *int
	DayOfMonth     *int
	SpecificDates  []time.Time
	MonthDayParity *MonthDayParity
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
