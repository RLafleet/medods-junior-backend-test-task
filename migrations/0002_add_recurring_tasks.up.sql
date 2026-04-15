CREATE TABLE IF NOT EXISTS task_templates (
	id               BIGSERIAL PRIMARY KEY,
	title            TEXT NOT NULL,
	description      TEXT NOT NULL DEFAULT '',
	is_active        BOOLEAN NOT NULL DEFAULT TRUE,
	recurrence_type  TEXT NOT NULL,
	start_date       DATE NOT NULL,
	timezone         TEXT NOT NULL DEFAULT 'UTC',
	every_n_days     INTEGER,
	day_of_month     INTEGER,
	specific_dates   DATE[],
	month_day_parity TEXT,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tasks
	ADD COLUMN IF NOT EXISTS template_id   BIGINT REFERENCES task_templates(id),
	ADD COLUMN IF NOT EXISTS scheduled_for DATE;

CREATE UNIQUE INDEX IF NOT EXISTS uq_tasks_template_scheduled
	ON tasks (template_id, scheduled_for)
	WHERE template_id IS NOT NULL;
