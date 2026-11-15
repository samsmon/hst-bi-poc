select count(*) as tickets, sum(total_price) / 1e12 as gross_t, sum(net_revenue) / 1e12 as net_t,
       avg(is_refunded::int) as refund_rate, avg(lead_time_days) as lead_days
from public.fact_ticket_sales_wide
