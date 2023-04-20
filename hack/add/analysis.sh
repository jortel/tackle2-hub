#!/bin/bash

host="${HOST:-localhost:8080}"
app="${1:-1}"

curl -X POST ${host}/applications/${app}/analyses \
  -H 'Content-Type:application/x-yaml' \
  -H 'Accept:application/x-yaml' -d \
'
rulesets:
- name: Main
  description: Testing.
  technologies:
  - name: TechA
  - name: TechB
  - name: TechC
  - name: TechD
  - name: TechE
  - name: TechS
    source: true
  issues:
  - ruleid: Main.001
    description: This is a test.
    category: warning
    effort: 10
    incidents:
    - uri: http://thing.com/file:1
      message: Thing happend line:1
      facts:
        factA: 1.A
        factB: 1.B
    - uri: http://thing.com/file:2
      message: Thing happend line:2
      facts:
        factA: 1.C
        factB: 1.D
    - uri: http://thing.com/file:3
      message: Thing happend line:3
      facts:
        factA: 1.E
        factB: 1.F
  - ruleid: Main.002
    description: This is a test.
    category: warning
    effort: 20
    incidents:
    - uri: http://thing.com/file:10
      message: Thing happend line:10
      facts:
        factA: 2.A
        factB: 2.B
    - uri: http://thing.com/file:20
      message: Thing happend line:20
      facts:
        factA: 2.C
        factB: 2.D
    - uri: http://thing.com/file:30
      message: Thing happend line:30
      facts:
        factA: 2.E
        factB: 2.F
  - ruleid: Main.003
    description: This is a test.
    category: warning
    effort: 10
    incidents:
    - uri: http://thing.com/file:10
      message: Thing happend line:10
      facts:
        factA: 2.A
        factB: 2.B
    - uri: http://thing.com/file:20
      message: Thing happend line:20
      facts:
        factA: 2.C
        factB: 2.D
    - uri: http://thing.com/file:30
      message: Thing happend line:30
      facts:
        factA: 2.E
        factB: 2.F
dependencies:
- name: github.com/libA
  version: 1.0
- name: github.com/libB
  version: 2.0
- name: github.com/libC
  version: 3.0
- name: github.com/libD
  version: 5.0
- name: github.com/libE
  version: 6.0
- name: github.com/libE
  indirect: true
  version: 7.0
'

