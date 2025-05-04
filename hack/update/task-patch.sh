#!/bin/bash

host="${HOST:-localhost:8080}"
id="${1:-1}"

curl -X PATCH ${host}/tasks/${id} \
  -H 'Content-Type:application/x-yaml' \
  -H 'Accept:application/x-yaml' \
 -d \
"
policy:
  preemptEnabled: true
"

