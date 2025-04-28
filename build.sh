#!/bin/sh
set -e

git archive HEAD -o static/source.tgz

docker build -t tinymail-build .
container=$(docker create tinymail-build)
docker cp "$container":/tinymail/tinymail ./tinymail
docker rm "$container"
