-- DRAFT data contract for ML output. Not executed by P1 and not final: it is
-- sent to the ML owner first and adjusted to what their pipeline can produce.
-- The BI only reads these tables.

CREATE SCHEMA IF NOT EXISTS ml;

CREATE TABLE IF NOT EXISTS ml.model_run (
    run_id          text PRIMARY KEY,
    model_name      text        NOT NULL,
    model_version   text        NOT NULL,
    trained_until   date        NOT NULL,
    forecast_origin date        NOT NULL,
    params          jsonb,
    status          text        NOT NULL CHECK (status IN ('running', 'success', 'failed')),
    is_demo         boolean     NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ml.forecast (
    run_id         text    NOT NULL REFERENCES ml.model_run (run_id),
    target_name    text    NOT NULL,           -- e.g. ticket_sales
    series_key     text    NOT NULL,           -- 'all', a route, or a channel
    grain          text    NOT NULL CHECK (grain IN ('day', 'week', 'month')),
    target_date    date    NOT NULL,
    horizon_days   integer NOT NULL,
    yhat           double precision NOT NULL,
    yhat_lower     double precision,
    yhat_upper     double precision,
    interval_level numeric(3, 2),              -- e.g. 0.80, 0.95
    PRIMARY KEY (run_id, target_name, series_key, grain, target_date)
);
CREATE INDEX IF NOT EXISTS forecast_lookup
    ON ml.forecast (target_name, series_key, target_date, run_id);

CREATE TABLE IF NOT EXISTS ml.backtest (
    run_id      text NOT NULL REFERENCES ml.model_run (run_id),
    target_name text NOT NULL,
    series_key  text NOT NULL,
    target_date date NOT NULL,
    yhat        double precision NOT NULL,
    actual      double precision NOT NULL,
    error       double precision NOT NULL,
    PRIMARY KEY (run_id, target_name, series_key, target_date)
);

CREATE TABLE IF NOT EXISTS ml.metric (
    run_id text NOT NULL REFERENCES ml.model_run (run_id),
    metric text NOT NULL,                      -- WAPE, MAE, MAPE
    scope  text NOT NULL,                      -- 'all' or a series key
    value  double precision NOT NULL,
    PRIMARY KEY (run_id, metric, scope)
);

CREATE TABLE IF NOT EXISTS ml.prediction_entity (
    run_id     text NOT NULL REFERENCES ml.model_run (run_id),
    task       text NOT NULL,                  -- churn, late_payment, discount
    entity_id  text NOT NULL,
    score      double precision NOT NULL,
    label_pred text,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, task, entity_id)
);
