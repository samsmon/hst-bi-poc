#!/bin/sh
# Build an Evidence site with our sources and page, then serve the static
# build. Timings go to /work/timings.txt.
#
# The template is pinned to the last open-source release (2026-02-06,
# @evidence-dev/evidence 40.1.8). Later template commits target the hosted
# Evidence Studio CLI, which needs a login and is not self-hostable this way.
set -e
TEMPLATE_REF=4b13d42e
APP=app-$TEMPLATE_REF
cd /work
t() { echo "$1 $(date +%s)" >> /work/timings.txt; }
: > /work/timings.txt
if [ ! -d "$APP" ]; then npx --yes degit "evidence-dev/template#$TEMPLATE_REF" "$APP"; fi
cd "$APP"
mkdir -p sources pages
find pages -mindepth 1 -delete
find sources -mindepth 1 -maxdepth 1 ! -name hst -exec mv {} /tmp/ \;
cp -r /src/sources/hst sources/
cp /src/pages/index.md pages/
t install_start; npm install --no-audit --no-fund; t install_end
t sources_start; npm run sources; t sources_end
t build_start; npm run build; t build_end
du -sk build | cut -f1 > /work/build_kb.txt
exec npx --yes serve -l 3000 build
