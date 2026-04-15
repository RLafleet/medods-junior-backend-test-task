# Task Service

Сервис для управления задачами с HTTP API на Go.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Причина в том, что SQL-файлы из `migrations/` монтируются в `docker-entrypoint-initdb.d` и применяются только при инициализации пустого data volume.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

Базовый префикс API:

```text
/api/v1
```

### Tasks

- `POST /api/v1/tasks` — create a task
- `GET /api/v1/tasks` — list all tasks (also triggers recurring task generation)
- `GET /api/v1/tasks/{id}` — get task by id
- `PUT /api/v1/tasks/{id}` — update task
- `DELETE /api/v1/tasks/{id}` — delete task instance

### Templates (recurring task rules)

- `POST /api/v1/templates` — create a recurring task template
- `GET /api/v1/templates` — list all templates
- `GET /api/v1/templates/{id}` — get template by id
- `PUT /api/v1/templates/{id}` — update template (affects only future generation)
- `DELETE /api/v1/templates/{id}` — deactivate template (sets `is_active = false`)

### Example: create a daily recurring template

```json
POST /api/v1/templates
{
  "title": "Daily standup",
  "description": "Write standup notes",
  "recurrence_type": "daily",
  "start_date": "2025-01-01",
  "timezone": "Europe/Moscow",
  "every_n_days": 1
}
```

### Example: create a monthly recurring template

```json
POST /api/v1/templates
{
  "title": "Monthly report",
  "recurrence_type": "monthly",
  "start_date": "2025-01-01",
  "timezone": "UTC",
  "day_of_month": 15
}
```

### Example: create a specific-dates template

```json
POST /api/v1/templates
{
  "title": "Quarterly review",
  "recurrence_type": "specific_dates",
  "start_date": "2025-01-01",
  "timezone": "UTC",
  "specific_dates": ["2025-01-15", "2025-04-15", "2025-07-15", "2025-10-15"]
}
```

### Example: create an even/odd days template

```json
POST /api/v1/templates
{
  "title": "Every odd day",
  "recurrence_type": "even_odd",
  "start_date": "2025-01-01",
  "timezone": "UTC",
  "month_day_parity": "odd"
}
```

### Recurrence types

| `recurrence_type`  | Required fields       | Description                                    |
|--------------------|-----------------------|------------------------------------------------|
| `daily`            | `every_n_days`        | Every N-th day from `start_date`               |
| `monthly`          | `day_of_month`        | Specific day of month (1–30)                   |
| `specific_dates`   | `specific_dates`      | Explicit list of dates                         |
| `even_odd`         | `month_day_parity`    | Days with even or odd day-number in the month  |

---

## Recurring Tasks: Architecture and Assumptions

### Separation of concerns

Task templates (`task_templates` table) and concrete task instances (`tasks` table) are separate entities.

- A **template** holds the recurrence rule: type, schedule parameters, timezone, and whether it is active.
- A **task instance** is a concrete item a user works with. It has a `status`, and optionally a `template_id` and `scheduled_for` date.

Manual tasks (created via `POST /api/v1/tasks`) have `template_id = NULL`.

### When tasks are generated

Tasks from active templates are generated **on demand** when `GET /api/v1/tasks` is called. The service generates task instances for all active templates from today up to **today + 30 days** (forward horizon).

There is no background worker or cron. Generation is idempotent: calling it multiple times produces no duplicates.

### Duplicate protection

A partial unique index on the `tasks` table prevents duplicates:

```sql
CREATE UNIQUE INDEX uq_tasks_template_scheduled
  ON tasks (template_id, scheduled_for)
  WHERE template_id IS NOT NULL;
```

The insert uses `ON CONFLICT ... DO NOTHING`, so repeated generation calls are safe.

### Assumptions

**daily — counting from start_date**
The first occurrence is `start_date` itself. Subsequent occurrences are `start_date + k * every_n_days` days. The sequence is always aligned to `start_date`, not to the current date.

**timezone**
Each template stores a `timezone` field (IANA name, e.g. `Europe/Moscow`). Date boundaries for monthly and even/odd recurrence are evaluated in that timezone. Defaults to `UTC` if not provided.

**monthly day 30 in shorter months**
If `day_of_month = 30` and the month has fewer than 30 days (February always; April, June, September, November in any non-leap year have 30 days so are fine), that month's occurrence is **skipped entirely**. No rescheduling to the last day of the month is performed.

**past dates in specific_dates**
Allowed. If a date from `specific_dates` falls before today, a task instance for that date will be generated (if the template was active at generation time) with status `new`. The user is responsible for handling past-dated tasks.

**even/odd definition**
Even means the calendar day number in the month is divisible by 2 (2, 4, 6, …). Odd means it is not (1, 3, 5, …). The parity field `month_day_parity` accepts `"even"` or `"odd"`.

**forward generation horizon**
30 calendar days from the date of the `GET /api/v1/tasks` request.

**template updates**
Updating a template (title, description, recurrence parameters) affects only **future generation**. Already-created task instances are never modified retroactively. Their title, description, and schedule remain as they were at the time of creation.

**deleting a template**
`DELETE /api/v1/templates/{id}` sets `is_active = false` (soft delete). Existing task instances linked to the template are not removed. To fully hide past tasks, delete each task instance individually.

**deleting a task instance**
`DELETE /api/v1/tasks/{id}` permanently removes that single task instance regardless of whether it was created from a template. If the template is still active, the task will be regenerated on the next `GET /api/v1/tasks` call.

---

## Running tests

```bash
go test ./...
```

Tests for the recurrence use case are in `internal/usecase/template/service_test.go` and cover:
- `daily` — stride and alignment from `start_date`
- `monthly` — normal month, `day_of_month = 30` in February (skipped), in October (included)
- `specific_dates` — filtering by generation range
- `even_odd` — even and odd parity
- Validation: `every_n_days <= 0`, `day_of_month` outside 1–30, empty `specific_dates`, duplicate `specific_dates`, invalid timezone
- Idempotency: repeated `GenerateUpTo` calls do not fail
- Template update: already-created tasks are not retroactively changed
