#!/bin/sh
set -e

echo "Updating yt-dlp..."
yt-dlp -U || echo "yt-dlp update failed, continuing with existing version..."

exec ./govd "$@"
