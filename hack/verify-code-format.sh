#!/usr/bin/env bash

FILES=$(gofmt -s -l pkg cmd test)

if [ -n "$FILES" ]; then
    echo You have go format errors in the below files, please run "gofmt -s -w pkg cmd"
    echo $FILES
    exit 1
fi

FILES=$(goimports -e -l -local=github.com/openshift/backup pkg cmd)

if [ -n "$FILES" ]; then
    echo You have go import errors in the below files, please run "goimports -e -w -local=github.com/openshift/backup pkg cmd"
    echo $FILES
    exit 1
fi
