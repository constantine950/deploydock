#!/bin/sh
# Certbot auto-renew script — run via cron every 12 hours
# Add to crontab: 0 */12 * * * /opt/deploydock/scripts/certbot-renew.sh

set -e

echo "[$(date)] running certbot renew..."
certbot renew --quiet --deploy-hook "nginx -s reload"
echo "[$(date)] certbot renew complete"