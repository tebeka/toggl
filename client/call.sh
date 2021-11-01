#!/bin/bash

set -e

api_token=$(jq -r .api_token ~/.togglrc)
workspace=$(jq -r .workspace ~/.togglrc)
base_url=https://api.track.toggl.com/api/v8
# url=${base_url}/time_entries/current
url=${base_url}/workspaces/${workspace}/projects


curl -f -u ${api_token}:api_token $url
