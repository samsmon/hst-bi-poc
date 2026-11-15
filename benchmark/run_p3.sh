#!/bin/bash
# Phase P3 runner. Runs on the host from the benchmark folder, next to a
# ../.env that holds BI_READER_PASSWORD. Writes everything to results/.
# Sequential on purpose: the database is shared with other services.
set -u
cd "$(dirname "$0")"
mkdir -p results
ENVF=../.env
add() { grep -q "^$1=" "$ENVF" || printf '%s=%s\n' "$1" "$2" >> "$ENVF"; }
add SUPERSET_SECRET_KEY "$(openssl rand -hex 32)"
add SUPERSET_ADMIN_PASSWORD "$(openssl rand -hex 16)"
add BIND_ADDR "$(tailscale ip -4 | head -1)"
set -a; . "$ENVF"; set +a
PGA="docker exec -i shared-postgres psql -U admin -d hst_dataset -X -v ON_ERROR_STOP=1"
DC="docker compose --env-file $ENVF -f compose.benchmark.yml"
MEM() { docker stats --no-stream --format '{{.Name}} {{.MemUsage}} cpu={{.CPUPerc}}' "$1"; }
wait_http() { # url, tries of 5 s, optional container that must stay running
  for _ in $(seq 1 "$2"); do
    curl -sf "$1" > /dev/null && return 0
    [ -z "${3:-}" ] || [ "$(docker inspect -f '{{.State.Running}}' "$3" 2>/dev/null)" = "true" ] || return 1
    sleep 5
  done
  return 1
}
S=$(date +%s)

echo "== matview" > results/00_steps.txt
T0=$(date +%s)
$PGA < 03_bi_matviews.sql > results/01_matview.txt 2>&1
echo "matview_exit $? seconds $(( $(date +%s) - T0 ))" >> results/00_steps.txt
$PGA -A -t -c "select count(*), pg_size_pretty(pg_total_relation_size('bi.mv_monthly_sales')) from bi.mv_monthly_sales" >> results/01_matview.txt 2>&1

echo "== postgres timings" >> results/00_steps.txt
: > results/02_postgres.txt
while IFS=$'\t' read -r name sql; do
  for run in 1 2; do
    ms=$(docker exec -e PGPASSWORD="$BI_READER_PASSWORD" shared-postgres psql -U bi_reader -h 127.0.0.1 -d hst_dataset -X -q -o /dev/null -c '\timing on' -c "$sql" 2>&1 | grep -E 'Time:|ERROR' | head -1)
    printf '%s\trun%s\t%s\n' "$name" "$run" "$ms" >> results/02_postgres.txt
  done
done < queries.tsv

echo "== superset" >> results/00_steps.txt
T0=$(date +%s)
$DC up -d --build superset > results/03_superset_up.txt 2>&1
wait_http "http://$BIND_ADDR:8611/health" 120 hst-bi-superset
echo "superset_ready_exit $? seconds $(( $(date +%s) - T0 ))" >> results/00_steps.txt
sleep 20
MEM hst-bi-superset > results/04_superset_mem_idle.txt 2>&1
python3 bench_superset.py "http://$BIND_ADDR:8611" queries.tsv > results/05_superset_queries.txt 2>&1
MEM hst-bi-superset > results/06_superset_mem_after.txt 2>&1
for p in /login/ /sqllab/; do
  curl -s -o /dev/null -w "$p %{http_code} %{time_total}s %{size_download}B\n" "http://$BIND_ADDR:8611$p"
done > results/07_superset_pages.txt 2>&1
docker images --format '{{.Repository}}:{{.Tag}} {{.Size}}' | grep -i superset > results/08_superset_image.txt 2>&1
docker logs --tail 40 hst-bi-superset > results/08_superset_log_tail.txt 2>&1
$DC stop superset >> results/03_superset_up.txt 2>&1

echo "== evidence" >> results/00_steps.txt
T0=$(date +%s)
$DC up -d evidence > results/09_evidence_up.txt 2>&1
( while [ "$(docker inspect -f '{{.State.Running}}' hst-bi-evidence 2>/dev/null)" = "true" ]; do
    docker stats --no-stream --format '{{.MemUsage}}' hst-bi-evidence | cut -d/ -f1
    sleep 10
  done > results/10_evidence_mem_samples.txt 2>&1 ) &
SAMPLER=$!
wait_http "http://$BIND_ADDR:8612/" 240 hst-bi-evidence
echo "evidence_ready_exit $? seconds $(( $(date +%s) - T0 ))" >> results/00_steps.txt
docker logs --tail 80 hst-bi-evidence > results/11_evidence_log_tail.txt 2>&1
docker exec hst-bi-evidence sh -c 'cat /work/timings.txt; echo build_kb $(cat /work/build_kb.txt)' > results/12_evidence_timings.txt 2>&1
for _ in 1 2 3; do
  curl -s -o /dev/null -w "/ %{http_code} %{time_total}s %{size_download}B\n" "http://$BIND_ADDR:8612/"
done > results/13_evidence_pages.txt 2>&1
MEM hst-bi-evidence > results/14_evidence_mem_serving.txt 2>&1
$DC stop evidence >> results/09_evidence_up.txt 2>&1
kill "$SAMPLER" 2>/dev/null
echo "total_seconds $(( $(date +%s) - S ))" >> results/00_steps.txt
echo DONE >> results/00_steps.txt
