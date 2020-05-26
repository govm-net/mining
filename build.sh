#!/bin/bash

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o mining_windows.exe
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mining_linux
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o mining_mac
folder=govm_mining
# echo $folder "$folder"
rm $folder -rf
mkdir $folder
mv mining_windows.exe $folder
mv mining_linux $folder
mv mining_mac $folder
cp conf.json $folder
tar zcvf "$folder"_$(date +'%Y%m%d_%H%M%S').tar.gz $folder
rm $folder -rf
echo Enter to exit
read k
