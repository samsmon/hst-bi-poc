#!/bin/bash
# Evidence part of the P3 benchmark, on its own (used after the first attempt
# hit the Evidence Studio template). Same outputs as run_p3.sh steps 09 to 14.
set -u
cd "$(dirname "$0")"
mkdir -p results
ENVF=../.env
set -a; . "$ENVF"; set +a
DC="docker compose --env-file $ENVF -f compose.benchmark.yml"
T0=$(date +%s)
$DC up -d --force-recreate evidence > results/09_evidence_up.txt 2>&1
( while [ "$(docker inspect -f '{{.State.Running}}' hst-bi-evidence 2>/dev/null)" = "true" ]; do
    docker stats --no-stream --format '{{.MemUsage}}' hst-bi-evidence | cut -d/ -f1
    sleep 10
  done > results/10_evidence_mem_samples.txt 2>&1 ) &
SAMPLER=$!
READY=1
for _ in $(seq 1 300); do
  curl -sf "http://$BIND_ADDR:8612/" > /dev/null && { READY=0; break; }
  [ "$(docker inspect -f '{{.State.Running}}' hst-bi-evidence 2>/dev/null)" = "true" ] || break
  sleep 5
done
echo "evidence_ready_exit $READY seconds $(( $(date +%s) - T0 ))" > results/00_steps_evidence.txt
docker inspect -f 'oom={{.State.OOMKilled}} exit={{.State.ExitCode}} running={{.State.Running}}' hst-bi-evidence >> results/00_steps_evidence.txt 2>&1
docker logs hst-bi-evidence 2>&1 | grep -v 'HTTP \|Returned ' | tail -60 > results/11_evidence_log_tail.txt
docker exec hst-bi-evidence sh -c 'cat /work/timings.txt; echo build_kb $(cat /work/build_kb.txt)' > results/12_evidence_timings.txt 2>&1
for _ in 1 2 3; do
  curl -s -o /dev/null -w "/ %{http_code} %{time_total}s %{size_download}B\n" "http://$BIND_ADDR:8612/"
done > results/13_evidence_pages.txt 2>&1
docker stats --no-stream --format '{{.Name}} {{.MemUsage}}' hst-bi-evidence > results/14_evidence_mem_serving.txt 2>&1
$DC stop evidence >> results/09_evidence_up.txt 2>&1
kill "$SAMPLER" 2>/dev/null
echo DONE >> results/00_steps_evidence.txt
