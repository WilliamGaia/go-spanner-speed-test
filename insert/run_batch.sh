#!/bin/bash

command="/Users/wangyuguang/Documents/vsop_spanner_test/insert/inserter.go"
batchsize="250"
num_instances=20
cmd_rounds=200

for ((i=0; i<=$cmd_rounds; i++))
do
    for ((j=1; j<=$num_instances; j++))
    do
        go run $command $batchsize & 
    done
    wait
    echo "All instances completed." 
done
echo "All insert jobs completed." 