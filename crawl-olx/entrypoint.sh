#!/bin/bash
set -e

# Start Xvfb
rm -f /tmp/.X99-lock
Xvfb :99 -screen 0 1024x768x24 &

export DISPLAY=:99

sleep 5

# Make Docker env variables available to cron
cat > /etc/profile.d/container-env.sh <<EOF
export SPREED_SHEET_AUTH="$SPREED_SHEET_AUTH"
export SPREED_SHEET_ID="$SPREED_SHEET_ID"
export RABBITMQ_SERVER="$RABBITMQ_SERVER"
export RABBITMQ_PORT="$RABBITMQ_PORT"
export RABBITMQ_DEFAULT_USER="$RABBITMQ_DEFAULT_USER"
export RABBITMQ_DEFAULT_PASS="$RABBITMQ_DEFAULT_PASS"
export RABBITMQ_DEFAULT_VHOST="$RABBITMQ_DEFAULT_VHOST"
export TARGET_WA_MESSAGE="$TARGET_WA_MESSAGE"
export DISPLAY="$DISPLAY"
EOF

# ----- Run once immediately -----
echo "Starting initial OLX crawl $(date)"
. /etc/profile.d/container-env.sh
/usr/local/bin/php /app/CrawlDataOLX.php

# ----- Install cron job -----
cat > /etc/cron.d/olx <<'EOF'
0 */6 * * * root . /etc/profile.d/container-env.sh && echo "Starting scheduled OLX crawl $(date)" >> /proc/1/fd/1 2>> /proc/1/fd/2 && /usr/local/bin/php /app/CrawlDataOLX.php >> /proc/1/fd/1 2>> /proc/1/fd/2
EOF

chmod 0644 /etc/cron.d/olx

# Load the crontab
crontab /etc/cron.d/olx

echo "Cron installed."

exec "$@"