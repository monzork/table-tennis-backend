#!/bin/sh
# Calls the app's /health/storage route, which pings Supabase Storage so the
# free project isn't paused for inactivity. No secrets needed on this machine.
# Usage: APP_URL=https://your-app.onrender.com ./supabase-keepalive.sh
set -eu

# Wait a random 0..JITTER_SECONDS (default 12h) so pings don't land at a fixed time.
sleep "$(awk -v max="${JITTER_SECONDS:-43200}" 'BEGIN { srand(); print int(rand() * max) }')"

code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 60 "${APP_URL:?set APP_URL}/health/storage")
echo "$(date -Is) keepalive: HTTP $code"
[ "$code" = 200 ]
