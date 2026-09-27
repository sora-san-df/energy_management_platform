CREATE TABLE IF NOT EXISTS meters (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    meter_id    TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    location    TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'PENDING',
    created_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS readings (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    meter_id         TEXT    NOT NULL,
    timestamp        TEXT    NOT NULL,
    consumption_kwh  REAL    NOT NULL,
    voltage_v        REAL    NOT NULL,
    current_a        REAL    NOT NULL,
    power_factor     REAL    NOT NULL,
    status           TEXT    NOT NULL DEFAULT 'OK',
    UNIQUE (meter_id, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_readings_meter_ts ON readings (meter_id, timestamp);

CREATE TABLE IF NOT EXISTS events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    meter_id     TEXT    NOT NULL,
    timestamp    TEXT    NOT NULL,
    type         TEXT    NOT NULL,
    description  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_meter ON events (meter_id, timestamp);

CREATE TABLE IF NOT EXISTS analyses (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    status              TEXT    NOT NULL,
    scope               TEXT    NOT NULL DEFAULT '[]',
    steps               TEXT    NOT NULL DEFAULT '[]',
    current_step        TEXT    NOT NULL DEFAULT '',
    meters_analyzed     INTEGER NOT NULL DEFAULT 0,
    anomalies_detected  INTEGER NOT NULL DEFAULT 0,
    high_priority       INTEGER NOT NULL DEFAULT 0,
    avg_confidence      REAL    NOT NULL DEFAULT 0,
    ai_provider         TEXT    NOT NULL DEFAULT '',
    ai_status           TEXT    NOT NULL DEFAULT '',
    summary             TEXT    NOT NULL DEFAULT '',
    error               TEXT    NOT NULL DEFAULT '',
    duration_ms         INTEGER NOT NULL DEFAULT 0,
    started_at          TEXT    NOT NULL,
    finished_at         TEXT
);

CREATE TABLE IF NOT EXISTS anomalies (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    analysis_id         INTEGER NOT NULL REFERENCES analyses (id),
    meter_id            TEXT    NOT NULL,
    detected_at         TEXT    NOT NULL,
    type                TEXT    NOT NULL,
    severity            TEXT    NOT NULL,
    confidence          REAL    NOT NULL,
    reason              TEXT    NOT NULL,
    recommended_action  TEXT    NOT NULL,
    status              TEXT    NOT NULL,
    source              TEXT    NOT NULL,
    ai_error            TEXT    NOT NULL DEFAULT '',
    ai_insights         TEXT    NOT NULL DEFAULT '[]',
    baseline_daily_kwh  REAL    NOT NULL DEFAULT 0,
    current_daily_kwh   REAL    NOT NULL DEFAULT 0,
    variation_pct       REAL    NOT NULL DEFAULT 0,
    window_start        TEXT,
    window_end          TEXT,
    affected_variables  TEXT    NOT NULL DEFAULT '[]',
    evidence            TEXT    NOT NULL DEFAULT '[]',
    related_event       TEXT,
    created_at          TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_anomalies_meter ON anomalies (meter_id);
