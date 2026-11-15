-- Pre-aggregation for dashboards. Run as the table owner against hst_dataset.
-- One small cube: month x season x channel x origin x destination x seat class.
-- Undo: DROP SCHEMA bi CASCADE;
-- Refresh after the fact table changes: REFRESH MATERIALIZED VIEW bi.mv_monthly_sales;

CREATE SCHEMA IF NOT EXISTS bi;

CREATE MATERIALIZED VIEW IF NOT EXISTS bi.mv_monthly_sales AS
SELECT date_trunc('month', travel_date)::date AS month,
       season_type,
       sales_channel_name,
       route_name,
       origin_station_name,
       destination_station_name,
       seat_class,
       count(*)                          AS tickets,
       sum(total_price)::bigint          AS gross_revenue,
       sum(net_revenue)::bigint          AS net_revenue,
       sum(coalesce(refund_amount, 0))::bigint AS refund_amount,
       sum(is_refunded::int)             AS refunded_tickets,
       sum(lead_time_days)::bigint       AS lead_time_days_sum
FROM public.fact_ticket_sales_wide
GROUP BY 1, 2, 3, 4, 5, 6, 7;

CREATE INDEX IF NOT EXISTS mv_monthly_sales_month ON bi.mv_monthly_sales (month);

GRANT USAGE ON SCHEMA bi TO bi_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA bi TO bi_reader;
ALTER DEFAULT PRIVILEGES IN SCHEMA bi GRANT SELECT ON TABLES TO bi_reader;
