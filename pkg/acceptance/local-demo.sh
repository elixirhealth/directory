#!/usr/bin/env bash

set -eou pipefail
#set -x  # useful for debugging

docker_cleanup() {
    echo "cleaning up existing network and containers..."
    CONTAINERS='libri|directory'
    docker ps | grep -E ${CONTAINERS} | awk '{print $1}' | xargs -I {} docker stop {} || true
    docker ps -a | grep -E ${CONTAINERS} | awk '{print $1}' | xargs -I {} docker rm {} || true
    docker network list | grep ${CONTAINERS} | awk '{print $2}' | xargs -I {} docker network rm {} || true
}

# optional settings (generally defaults should be fine, but sometimes useful for debugging)
DIRECTORY_LOG_LEVEL="${DIRECTORY_LOG_LEVEL:-INFO}"  # or DEBUG
DIRECTORY_TIMEOUT="${DIRECTORY_TIMEOUT:-5}"  # 10, or 20 for really sketchy network

# local and filesystem constants
LOCAL_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# container command constants
DIRECTORY_IMAGE="gcr.io/elixir-core-prod/directory:snapshot" # develop

echo
echo "cleaning up from previous runs..."
docker_cleanup

echo
echo "creating directory docker network..."
docker network create directory

echo
echo "starting directory..."
port=10100
name="directory-0"
docker run --name "${name}" --net=directory -d -p ${port}:${port} ${DIRECTORY_IMAGE} \
    start \
    --logLevel "${DIRECTORY_LOG_LEVEL}" \
    --serverPort ${port}
directory_addrs="${name}:${port}"
directory_containers="${name}"

echo
echo "testing directory health..."
docker run --rm --net=directory ${DIRECTORY_IMAGE} test health \
    --directories "${directory_addrs}" \
    --logLevel "${DIRECTORY_LOG_LEVEL}"

echo
echo "testing directory ..."
docker run --rm --net=directory ${DIRECTORY_IMAGE} test io \
    --directories "${directory_addrs}" \
    --logLevel "${DIRECTORY_LOG_LEVEL}"

echo
echo "cleaning up..."
docker_cleanup

echo
echo "All tests passed."
