select date_trunc('month', travel_date)::date as month, season_type,
       count(*) as tickets, sum(net_revenue) / 1e9 as net_revenue_b
from public.fact_ticket_sales_wide group by 1, 2
