select origin_station_name || ' - ' || destination_station_name as route,
       count(*) as tickets, sum(net_revenue) / 1e9 as net_revenue_b,
       avg(is_refunded::int) as refund_rate
from public.fact_ticket_sales_wide group by 1 order by 2 desc limit 10
