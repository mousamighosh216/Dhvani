#!/usr/bin/env bash
# Ingest the generated sample songs into a running server, then try every clip.
#   make samples
#   ./scripts/seed.sh [http://localhost:8080]
set -euo pipefail
API="${1:-http://localhost:8080}"
cd "$(dirname "$0")/.."

for f in samples/songs/song*.wav; do
  n=$(basename "$f" .wav | tr -d 'a-z')
  curl -fsS -o /dev/null -F "audio=@$f" -F "title=Synth Song $n" -F "artist=The Synths" \
       -F "album=Test Tones" -F "release_date=2024-01-0$n" "$API/api/v1/songs"
  echo "ingested $f"
done

echo; echo "Recognition results:"
for f in samples/clips/*.wav; do
  printf "%-32s " "$(basename "$f")"
  curl -fsS -F "audio=@$f" "$API/api/v1/recognize" | python3 -c '
import sys, json
d = json.load(sys.stdin)["data"]
print(("MATCH  " + d["song"]["title"] + "  (conf %.2f, %d landmarks)" % (d["confidence"], d["votes"])) if d["matched"] else "no match")'
done
