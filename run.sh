#/bin/bash

for F in {1..100} ; do
  for X in {1..5} ; do
    time hcsvlab-consumer $F > /dev/null
  done
done
