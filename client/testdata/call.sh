#!/bin/bash
# Used for calling API from curl

set -e

api_token=$(jq -r .api_token ~/.togglrc)
workspace=$(jq -r .workspace ~/.togglrc)
base_url=https://api.track.toggl.com/api/v9
url=${base_url}/workspaces/${workspace}/time_entries

cat <<EOF > /tmp/start.json
{
    "duration":     -1,
    "start":        "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "created_with": "github.com/tebeka/toggl",
    "workspace_id": ${workspace}
}
EOF

curl -f \
    -u ${api_token}:api_token \
    -X POST \
    -d @/tmp/start.json \
    $url