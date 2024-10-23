#!/bin/bash
for k in {0..9}
do
  for j in {0..9}
  do
    for i in {0..9}
    do
      kubectl run p${k}${j}${i} -n h9 --image=nginx:latest
    done
  done
done
