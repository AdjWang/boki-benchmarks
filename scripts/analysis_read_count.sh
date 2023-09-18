#!/bin/bash
set -uo pipefail

# SEARCH_DIR=/tmp/boki-test/mnt/
# SEARCH_DIR=/home/ubuntu/boki-benchmarks/experiments/retwis/boki/results/respcount/con192-sync-sequential-respcount/fn_output
# SEARCH_DIR=/home/ubuntu/boki-benchmarks/experiments/retwis/boki/results/respcount/con192-async-sequential-respcount/fn_output
# SEARCH_DIR=/home/ubuntu/boki-benchmarks/experiments/retwis/boki/results/respcount/con192-sync-strong-respcount/fn_output
# SEARCH_DIR=/home/ubuntu/boki-benchmarks/experiments/retwis/boki/results/respcount/con192-async-strong-respcount/fn_output
# SEARCH_DIR=/home/ubuntu/boki-benchmarks/experiments/retwis/boki/results/respcount/con193-sync-strong-respcount-txn9010/fn_output
SEARCH_DIR=/home/ubuntu/boki-benchmarks/experiments/retwis/boki/results/respcount/con192-async-strong-respcount-txn9010/fn_output

echo "Append"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "Append" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadNext"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "ReadNext" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadPrev"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "ReadPrev" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadNextB"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "ReadNextB" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadSyncToData"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "ReadSyncToData" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadSyncToEOF"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "ReadSyncToEOF" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadLocalId"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "ReadLocalId" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadPrevAux"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "ReadPrevAux" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "ReadUnknown"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "ReadUnknown" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Aux"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "Aux" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Empty"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "Empty" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Other"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "Other" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Fifo"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "Fifo" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
echo "Shm"
find $SEARCH_DIR -name *.stderr | xargs grep "STAT" | grep "Shm" | egrep "[0-9]+ samples" -o | awk '{print $1}' | paste -sd+ | bc
