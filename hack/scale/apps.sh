#!/bin/bash

baseName="Test"
count=10

pid=$$
self=$(basename $0)
tmp=/tmp/${self}-${pid}

usage() {
  echo "Usage: ${self} <required> <options>"
  echo "-h help"
  echo "Required:"
  echo "  -u <URL>"
  echo "Options:"
  echo "  -b base name (Test)"
  echo "  -n count  (10)"
  echo "  -t tag"
  echo "  -o output"
}

while getopts "u:b:n:t:o:h" arg; do
  case $arg in
    u)
      host=$OPTARG
      ;;
    b)
      baseName=$OPTARG
      ;;
    n)
      count=$OPTARG
      ;;
    t)
      tag="- id: $OPTARG"
      ;;
    o)
      output=$OPTARG
      ;;
    h)
      usage
      exit 1
  esac
done

if [ -z "${host}"  ]
then
  echo "-u required."
  usage
  exit 0
fi

print() {
  if [ -n "${output}"  ]
  then
    echo -e "$@" >> ${output}
  else
    echo -e "$@"
  fi
}

echo
echo "Host:   ${host}"
echo "Name:   ${baseName}"
echo "Count:  ${count}"
echo
answer="y"
read -p "Continue[Y,n]: " answer
if [ "$answer" != "y" ]
then
  exit 0
fi


createApplications() {
for i in $(seq 1 ${count})
do
name="${baseName}-${i}"
d="
---
name: ${name}
description: ${name} Test application.
repository:
  kind: git
  url: https://github.com/WASdev/sample.daytrader7.git
#binary:
# group: konveyor.io
# artifact: test
# version: 1.0
#tags:
#${tag}
"
code=$(curl -kSs -o ${tmp} -w "%{http_code}" -X POST ${host}/applications -H 'Content-Type:application/x-yaml' -d "${d}")
if [ ! $? -eq 0 ]
then
  exit $?
fi
case ${code} in
  201)
   id=$(jq .id ${tmp})
   print "Application ${name} id=${id} - CREATED"
   ;;
 *)
   print "Create application ${name} - FAILED: ${code}."
   cat ${tmp}
   exit 1
esac
done
}

createApplications
