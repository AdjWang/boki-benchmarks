#!/bin/bash
set -uo pipefail

echo "Append"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "Append" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadNext"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "ReadNext" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadPrev"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "ReadPrev" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadNextB"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "ReadNextB" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadSyncTo"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "ReadSyncTo" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadLocalId"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "ReadLocalId" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadPrevAux"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "ReadPrevAux" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadUnknown"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "ReadUnknown" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Aux"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "Aux" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Empty"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "Empty" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Other"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "Other" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Fifo"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "Fifo" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Shm"
find /tmp/boki-test/mnt/ -name *.stderr | xargs grep "STAT" | grep "Shm" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
