#!/bin/sh

set -x
go build
go install
launchctl unload ~/Library/LaunchAgents/com.adaptivekind.markdown-reader-mcp.plist
launchctl load ~/Library/LaunchAgents/com.adaptivekind.markdown-reader-mcp.plist
