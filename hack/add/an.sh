#!/bin/bash

set -e
set -x

host="${HOST:-localhost:8080}"
appId="${1:-1}"
file="${2:-/tmp/manifest.yaml}"
tmp=/tmp/${self}-${pid}

#
# Post manifest.
code=$(curl -kSs -o ${tmp} -w "%{http_code}" -F "file=@${file};type=application/x-yaml" "http://${host}/files/manifest")
if [ ! $? -eq 0 ]
then
  exit $?
fi
case ${code} in
  201)
    manifestId=$(cat ${tmp}|jq .id)
    echo "manifest (file): ${name} posted. id=${manifestId}"
    ;;
  *)
    echo "manifest (file) post - FAILED: ${code}."
    cat ${tmp}
    exit 1
esac

#
# Post analysis.
d="
id: ${manifestId}
"
code=$(curl -kSs -o ${tmp} -w "%{http_code}" "${host}/applications/${appId}/analyses" -H "Content-Type:application/x-yaml" -d "${d}")
if [ ! $? -eq 0 ]
then
  exit $?
fi
case ${code} in
  201)
    id=$(cat ${tmp}|jq .id)
    echo "analysis: ${name} posted. id=${id}"
    ;;
  *)
    echo "analysis post  - FAILED: ${code}."
    cat ${tmp}
    exit 1
esac

