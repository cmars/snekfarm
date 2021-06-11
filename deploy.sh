#!/usr/bin/env bash
set -eux
TARGET=ubuntu@snekfarm.cmars.tech

docker build -t snekfarm .
docker save snekfarm:latest | ssh $TARGET "docker load"
ssh $TARGET 'docker rm --force $(docker ps --format "{{.ID}}" --filter name=snekfarm); docker run -d --restart always --name snekfarm -p 80:3000 snekfarm:latest'
