#!/bin/bash

host="${HOST:-localhost:8080}"

echo $HOST

# ID to delete (default:1)
id="${1:-1}"

curl -X DELETE ${host}/tasks/${id}

