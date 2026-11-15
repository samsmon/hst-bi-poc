---
title: HST overview (Evidence benchmark)
---

```sql kpi
select * from hst.kpi
```

<BigValue data={kpi} value=tickets fmt=num0 />
<BigValue data={kpi} value=gross_t title="Gross revenue (IDR T)" fmt=num3 />
<BigValue data={kpi} value=net_t title="Net revenue (IDR T)" fmt=num3 />
<BigValue data={kpi} value=refund_rate fmt=pct2 />
<BigValue data={kpi} value=lead_days title="Average lead time (days)" fmt=num1 />

## Net revenue by month and season type

```sql monthly
select month, season_type, net_revenue_b from hst.monthly_season order by month
```

<BarChart data={monthly} x=month y=net_revenue_b series=season_type yAxisTitle="IDR billions" />

## Channel mix

```sql channels
select sales_channel_name, tickets from hst.channel_mix order by tickets desc
```

<BarChart data={channels} x=sales_channel_name y=tickets swapXY=true />

## Top routes

```sql routes
select * from hst.top_routes
```

<DataTable data={routes} />
