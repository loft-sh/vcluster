#!/bin/bash

declare -a arr
for subdir in config/v*/; do
  arr+=("${subdir}")
  go run config/"${subdir#config/}"schema/main.go
done

declare -a arr2
arr2=($(sort -V <<<"${arr[*]}"))
latest="${arr2[-1]}"

cp "${latest}"chart/* chart/
