select sales_channel_name, count(*) as tickets
from public.fact_ticket_sales_wide group by 1
