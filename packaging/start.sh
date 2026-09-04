#!/bin/sh
cd "$(dirname "$0")"
echo "Starting the 3D Job Desk printer bridge. Leave this window open."
exec ./3d-job-desk-bridge
