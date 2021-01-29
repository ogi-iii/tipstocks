#! /bin/bash

# generate files
# ./tools/protoc.sh
./tools/ssl.sh

# build client & server apps
cd app/client
GOOS=linux GOARCH=amd64 go build -o linux-amd64/client
cd ../server
GOOS=linux GOARCH=amd64 go build -o linux-amd64/server
cd ../..

# build app
docker-compose build
