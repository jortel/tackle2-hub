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
  echo "  -o output"
}

while getopts "u:b:n:o:h" arg; do
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


createReport() {
taskId=$1
d="
---
status: Succeeded
activity:
- 'OptDir:    /addon/opt'
- 'SharedDir: /shared'
- 'CacheDir:  /cache'
- 'SourceDir: /shared/source'
- 'RuleDir:   /addon/rules'
- 'BinDir:    /shared/bin'
- 'M2Dir:     /cache/m2'
- Fetching application.
- '[CMD] Running: /usr/bin/ssh-agent -a /tmp/agent.1'
- '[CMD] /usr/bin/ssh-agent succeeded.'
- '[SSH] Agent started.'
- '[GIT] Cloning: https://github.com/WASdev/sample.daytrader7.git'
- '[FILE] Created /addon/.gitconfig.'
- '[CMD] Running: /usr/bin/git clone https://github.com/WASdev/sample.daytrader7.git /shared/source/sample'
- '[CMD] /usr/bin/git succeeded.'
- '[RULESET] fetching: id=1 (.discovery)'
- '[RULESET] fetching: id=24 (cloud-readiness)'
- '[RULESET] fetching (dep): id=20 (.technology-usage)'
- '[CMD] Running: /usr/bin/windup-shim convert --outputdir /addon/rules/converted /addon/rules'
- '[CMD] /usr/bin/windup-shim succeeded.'
- '[CMD] Running: /usr/bin/konveyor-analyzer --provider-settings /addon/opt/settings.yaml --output-file /addon/report.yaml --no-dependency-rules --rules /addon/rules/rulesets/1/rules --rules /addon/rules/rulesets/24/rules --rules /addon/rules/rulesets/20/rules --label-selector konveyor.io/target=cloud-readiness --dep-label-selector !konveyor.io/dep-source=open-source'
- '[CMD] /usr/bin/konveyor-analyzer succeeded.'
- '[CMD] Running: /usr/bin/konveyor-analyzer-dep --provider-settings /addon/opt/settings.yaml --output-file /addon/deps.yaml'
- '[CMD] /usr/bin/konveyor-analyzer-dep succeeded.'
- 'Analysis reported. duration: 9.769534ms'
- '[TAG] Tagging Application 7.'
- Facts updated.
- Done.
"
code=$(curl -kSs -o ${tmp} -w "%{http_code}" -X POST ${host}/tasks/${taskId}/report -H 'Content-Type:application/x-yaml' -d "${d}")
if [ ! $? -eq 0 ]
then
  exit $?
fi
case ${code} in
  201)
   id=$(jq .id ${tmp})
   print "TaskReport ${id} CREATED for: taskId=${taskId}"
   ;;
 *)
   print "Create (task) report for: taskId=${taskId} - FAILED: ${code}."
   cat ${tmp}
   exit 1
esac
}


createTask() {
appId=$1
appName=$2
d="
---
kind: analyzer
application:
  id: ${appId}
data:
  mode:
    binary: false
    withDeps: false
  rules:
    labels:
      included:
        - konveyor.io/target=testA
        - konveyor.io/target=testB
        - konveyor.io/target=testC
        - konveyor.io/target=testD
        - konveyor.io/target=testE
"
code=$(curl -kSs -o ${tmp} -w "%{http_code}" -X POST ${host}/tasks -H 'Content-Type:application/x-yaml' -d "${d}")
if [ ! $? -eq 0 ]
then
  exit $?
fi
case ${code} in
  201)
   id=$(jq .id ${tmp})
   print "Task ${id} CREATED for application: ${appName} id=${appId}"
   createReport ${id}
   ;;
 *)
   print "Create task for: appId=${appId} - FAILED: ${code}."
   cat ${tmp}
   exit 1
esac
}


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
tags:
- id: 16
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
   createTask ${id} ${name}
   ;;
 *)
   print "Create application ${name} - FAILED: ${code}."
   cat ${tmp}
   exit 1
esac
done
}

createApplications
