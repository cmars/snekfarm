#!/usr/bin/env bash
set -eux
docker build -t snekfarm .
docker save snekfarm:latest | ssh ubuntu@snekfarm.cmars.tech "docker load"
