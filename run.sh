#!/bin/sh
if (go build main.go); then
  ./main
else
  echo "ERROR: $?"
fi


