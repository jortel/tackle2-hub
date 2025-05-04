#!/bin/bash

host="${HOST:-localhost:8080}"
name="${1:Test}"
begin=${2:-1}
end=${3:-10}

for i in $(seq ${begin} ${end})
do
d="
---
name: ${name}${i}
description: ${name}${i} application.
repository:
  kind: git
  url: https://github.com/WASdev/sample.daytrader7.git
tags:
- id: 1
- id: 16
"
curl -X POST ${host}/applications \
  -H 'Content-Type:application/x-yaml' \
  -H 'Accept:application/x-yaml' \
 -d "${d}"
done
