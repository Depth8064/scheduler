CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'workstation')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workstations (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    station_type TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_workstation_access (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workstation_id UUID NOT NULL REFERENCES workstations(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, workstation_id)
);

CREATE TABLE IF NOT EXISTS jobboss_po_lines (
    id UUID PRIMARY KEY,
    external_po_number TEXT NOT NULL,
    external_part_number TEXT NOT NULL,
    external_item_number TEXT NOT NULL,
    status TEXT,
    required_date DATE,
    qty_required NUMERIC(14, 2) NOT NULL DEFAULT 0,
    raw_payload JSONB NOT NULL,
    external_last_mod_date TIMESTAMPTZ,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (external_po_number, external_part_number, external_item_number)
);

CREATE TABLE IF NOT EXISTS schedule_items (
    id UUID PRIMARY KEY,
    po_line_id UUID REFERENCES jobboss_po_lines(id) ON DELETE RESTRICT,
    workstation_id UUID REFERENCES workstations(id),
    sort_order INT,
    status TEXT NOT NULL CHECK (status IN ('unreleased', 'released', 'queued', 'in_progress', 'paused', 'blocked', 'completed')),
    job_type TEXT NOT NULL CHECK (job_type IN ('OPEN', 'KAN_BAN')),
    job_group TEXT,
    part_number TEXT NOT NULL,
    part_description TEXT,
    material_spec TEXT,
    external_job_number TEXT,
    receiving_reference TEXT,
    planned_start_date DATE,
    required_date DATE,
    target_date DATE,
    priority INT NOT NULL DEFAULT 0,
    planner_notes TEXT,
    qty_required NUMERIC(14, 2) NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS execution_progress (
    id UUID PRIMARY KEY,
    schedule_item_id UUID NOT NULL UNIQUE REFERENCES schedule_items(id) ON DELETE CASCADE,
    qty_complete NUMERIC(14, 2) NOT NULL DEFAULT 0,
    qty_remaining NUMERIC(14, 2) NOT NULL DEFAULT 0,
    scrap_delta NUMERIC(14, 2) NOT NULL DEFAULT 0,
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS execution_events (
    id UUID PRIMARY KEY,
    schedule_item_id UUID NOT NULL REFERENCES schedule_items(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    actor_id UUID REFERENCES users(id),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    event_payload JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS kanban_items (
    part_number TEXT PRIMARY KEY,
    part_description TEXT,
    units_per_assembly INT NOT NULL DEFAULT 0,
    monthly_qty INT NOT NULL DEFAULT 0,
    weekly_qty INT NOT NULL DEFAULT 0,
    bin_qty INT NOT NULL DEFAULT 0,
    total_bins INT NOT NULL DEFAULT 0,
    total_kanban_qty INT NOT NULL DEFAULT 0,
    bin_type_notes TEXT,
    dedicated_workstation_id UUID REFERENCES workstations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync_checkpoints (
    resource_name TEXT PRIMARY KEY,
    last_sync_at TIMESTAMPTZ,
    last_modified_max TIMESTAMPTZ,
    status TEXT NOT NULL,
    error_message TEXT,
    record_count INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sync_runs (
    id UUID PRIMARY KEY,
    resource_name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    record_count INT NOT NULL DEFAULT 0,
    error_summary TEXT
);

CREATE INDEX IF NOT EXISTS idx_schedule_items_workstation_sort_order
    ON schedule_items(workstation_id, sort_order);

CREATE INDEX IF NOT EXISTS idx_schedule_items_state
    ON schedule_items(state);

CREATE INDEX IF NOT EXISTS idx_execution_events_schedule_item
    ON execution_events(schedule_item_id, occurred_at);
