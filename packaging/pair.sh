#!/bin/sh
cd "$(dirname "$0")"
echo "Pair this computer with your 3D Job Desk."
echo "Get a pairing code from Printers on the website first."
exec ./3d-job-desk-bridge pair
