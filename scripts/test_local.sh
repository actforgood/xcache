#!/usr/bin/env bash

#
# This script runs integration tests locally.
# First argument (optional), the Redis setup, one of "single" / "failover" / "cluster", defaults to "single".
# Second argument (optional), specifies that benchmarks should be run, not tests.
#
# Example of usage of this script:
# ./path/to/scripts/test_local.sh
# or
# ./path/to/scripts/test_local.sh failover benchmark
#

REDIS_SETUP=single
if [ "$1" != "" ]; then
    if [ "$1" == "single" ] || [ "$1" == "failover" ] || [ "$1" == "cluster" ]; then
        REDIS_SETUP=$1
    fi
fi
RUN="test"
if [ "$2" != "" ]; then
    RUN="benchmark"
fi
SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
CHECK_SIGN="[\033[0;32m\xE2\x9C\x94\033[0;34m]"
CROSS_SIGN="[\xE2\x9D\x8C]"
REDIS_VERSIONS=( 6 7 )

DOCKER_REDIS6_IMAGE_VER=redis:6.2.18-alpine3.21
DOCKER_REDIS7_IMAGE_VER=redis:7.4.4-alpine3.21
DOCKER_TEST_RUNNER_IMAGE_NAME="xcache-${RUN}-runner"
DOCKER_NETWORK="xcache-network"
if [ "$REDIS_SETUP" == "cluster" ]; then
    DOCKER_NETWORK=host
fi

# debug prints the given debug message in blue color.
# Example: debug "Hello World".
debug() {
    printf "\033[0;34m>>> $@\033[0m\n"
}

# fatal prints the given error message in red color, 
# followed by error exit status.
# Example: fatal "Something bad happened"
fatal() {
    printf "\033[0;31m>>> ERR: $@\033[0m\n"
    exit 1
}

# checkCommands checks if given commands are available.
# Example: checkCommands "docker" "git"
checkCommands() {
    for cmd in "$@"; do
        if ! [ -x "$(command -v "${cmd}")" ]; then
            fatal "'${cmd}' is not available."
        fi
    done
}

# setUp prepares testing environment.
setUp() {
    tearDown

    createNetwork "$DOCKER_NETWORK"
    createImage "${SCRIPT_PATH}/../build/Dockerfile.${RUN}.local" "$DOCKER_TEST_RUNNER_IMAGE_NAME" "${SCRIPT_PATH}/.."
    for redisVer in "${REDIS_VERSIONS[@]}"; do
        redisImg="DOCKER_REDIS${redisVer}_IMAGE_VER"
        pullImage "${!redisImg}"
    done
}

# tearDown cleans up testing environment.
tearDown() {
    removeContainersByRegex "^xcache-.+"
    if [ "$DOCKER_NETWORK" != "host" ]; then
        removeNetwork "$DOCKER_NETWORK"
    fi
    removeImage "$DOCKER_TEST_RUNNER_IMAGE_NAME"
}

# pullImage pulls given docker image.
# Example: pullImage "redis"
pullImage() {
    debug "Pulling image ..."
    docker pull -q "$1"
}

# removeContainersByRegex stops and deletes container(s) that match(es) given regular expression.
# Example: removeContainersByRegex "redis"
removeContainersByRegex() {
    existing=$(docker container ls -a | awk '{print $NF}' | grep -E "$1")
    if [ "$existing" != "" ]; then
        debug "Removing containers ..."
        docker ps | awk '{print $NF}' | grep -E "$1" | xargs docker stop > /dev/null
        docker container ls -a | awk '{print $NF}' | grep -E "$1" | xargs docker rm
    fi
}

# removeNetwork deletes the provided docker network name, if it exists.
# Example: removeNetwork "some-network"
removeNetwork() {
    existing=$(docker network ls | awk '{print $2}' | grep -E "^${1}$")
    if [ "$existing" == "$1" ]; then
        debug "Removing network ..."
        docker network rm "$1"
    fi
}

# createNetwork creates the provided docker network, if it does not exist.
# Example: createNetwork "some-network"
createNetwork() {
    existing=$(docker network ls | awk '{print $2}' | grep -E "^${1}$")
    if [ "$existing" == "" ]; then
        debug "Creating network ..."
        docker network create "$1"
    fi
}

# removeImage deletes the given docker image.
# Example: removeImage "redis"
removeImage() {
    existing=$(docker image ls | awk '{print $1}' | grep -E "^${1}$")
    if [ "$existing" == "$1" ]; then
        debug "Removing image ..."
        docker image rm "$1"
    fi
}

# createImage builds an image.
# First argument is the full path to Dockerfile.
# Second argument is the image name.
# Third argument is the context path.
createImage() {
    debug "Creating ${RUN} runner image ..."
    docker build -q -f "$1" -t "$2" "$3"
}

# setUpRedisSingleInstance starts a single instance Redis container.
setUpRedisSingleInstance() {
    for redisVer in "${REDIS_VERSIONS[@]}"; do
        redisImg="DOCKER_REDIS${redisVer}_IMAGE_VER"
        redisContainer="xcache-redis${redisVer}-single"
        docker run -d \
            --name "$redisContainer" \
            --network "$DOCKER_NETWORK" \
            "${!redisImg}"
        checkRedisInstance "$redisContainer" 6379
    done
}

# setUpRedisFailover starts a master with 2 slaves, and 3 sentinels.
setUpRedisFailover() {
    for redisVer in "${REDIS_VERSIONS[@]}"; do
        redisImg="DOCKER_REDIS${redisVer}_IMAGE_VER"

        redisContainerMaster="xcache-redis${redisVer}-master"
        docker run -d \
            --name "$redisContainerMaster" \
            --network "$DOCKER_NETWORK" \
            "${!redisImg}"
        checkRedisInstance "$redisContainerMaster" 6379
        
        for replicaNo in 1 2; do
            redisContainerSlave="xcache-redis${redisVer}-replica-${replicaNo}"
            docker run -d \
                --name "$redisContainerSlave" \
                --network "$DOCKER_NETWORK" \
                "${!redisImg}" \
                redis-server --replicaof "$redisContainerMaster" 6379
            checkRedisInstance "$redisContainerSlave" 6379
        done

        for sentinelNo in 1 2 3; do
            cp "${SCRIPT_PATH}/../build/sentinel${redisVer}.conf.dist" "${SCRIPT_PATH}/../build/sentinel${redisVer}-${sentinelNo}.conf"
            redisContainerSentinel="xcache-redis${redisVer}-sentinel-${sentinelNo}"
            docker run -d \
                --name "$redisContainerSentinel" \
                --network "$DOCKER_NETWORK" \
                -v "${SCRIPT_PATH}/../build":/redis-conf/ \
                "${!redisImg}" \
                redis-server "/redis-conf/sentinel${redisVer}-${sentinelNo}.conf" --sentinel
            checkRedisInstance "$redisContainerSentinel" 26379
        done
        checkFailoverConfiguration "xcache-redis${redisVer}-sentinel-1" 26379
    done
}

# setUpRedisCluster starts a Redis Cluster with 3 masters and 3 slaves.
setUpRedisCluster() {
    for redisVer in "${REDIS_VERSIONS[@]}"; do
        redisImg="DOCKER_REDIS${redisVer}_IMAGE_VER"
        hosts=""
        basePort="${redisVer}000"
        for nodeNo in 1 2 3 4 5 6; do
            port=$(( basePort + nodeNo ))
            redisContainerClusterNode="xcache-redis${redisVer}-cluster-node-${nodeNo}"
            docker run -d \
                --name "$redisContainerClusterNode" \
                --network "$DOCKER_NETWORK" \
                "${!redisImg}" \
                redis-server --cluster-enabled yes --port $port
            checkRedisInstance "$redisContainerClusterNode" $port
            hosts="${hosts} 127.0.0.1:${port}"
        done
        port=$(( basePort + 1 ))
        reply="$(docker exec "xcache-redis${redisVer}-cluster-node-1" \
            redis-cli -p $port \
            --cluster create$hosts \
            --cluster-replicas 1 \
            --cluster-yes | tail -1)"
        if [[ "$reply" =~ "OK" ]]; then
            debug "Configuring cluster ${CHECK_SIGN}"
        else 
            fatal "Configuring cluster ${CROSS_SIGN} ($reply)"
        fi
    done
}

# runTests runs the tests/benchmarks through DOCKER_TEST_RUNNER_IMAGE_NAME.
# 1st argument is a string representing Redis6 address(es), comma separated.
# 2nd argument is a string representing the master name in case of 'sentinel' configuration for Redis6.
# 3rd argument is a string representing Redis7 address(es), comma separated.
# 4th argument is a string representing the master name in case of 'sentinel' configuration for Redis7.
runTests() {
    out="$(docker run \
        --name "$DOCKER_TEST_RUNNER_IMAGE_NAME" \
        --network "$DOCKER_NETWORK" \
        -e XCACHE_REDIS6_ADDRS="$1" \
        -e XCACHE_REDIS6_MASTER_NAME="$2" \
        -e XCACHE_REDIS7_ADDRS="$3" \
        -e XCACHE_REDIS7_MASTER_NAME="$4" \
        "$DOCKER_TEST_RUNNER_IMAGE_NAME")"
    printf "%s\n" "$out"
    if [[ "$out" =~ ok.+github ]]; then
        debug "Run ${RUN}s ${CHECK_SIGN}"
    else 
        debug "Run ${RUN}s ${CROSS_SIGN}"
    fi
}

# checkFailoverConfiguration verifies that 1 master with 2 slaves watched by 3 sentinels is properly configured.
# First argument is one of the sentinels' ip/hostname.
# Second argument is the port of that sentinel.
checkFailoverConfiguration() {
    sentinelInstance=$1
    sentinelPort=$2
    retryNo=0
    maxRetries=5
    conditionsMet=0
    debug "Checking sentinel configuration ..."
    while true ; do
        conditionsMet=0
        reply=$(docker exec "$sentinelInstance" redis-cli -p "$sentinelPort" sentinel master xcacheMaster 2> /dev/null)
        [[ $reply =~ ip[[:space:]](xcache-redis.-master)[[:space:]] ]] && masterIP=${BASH_REMATCH[1]}
        [[ $reply =~ port[[:space:]](6379)[[:space:]] ]] && masterPort=${BASH_REMATCH[1]}
        [[ $reply =~ num-slaves[[:space:]]([[:digit:]]{1})[[:space:]] ]] && numSlaves=${BASH_REMATCH[1]}
        [[ $reply =~ num-other-sentinels[[:space:]]([[:digit:]]{1})[[:space:]] ]] && numOtherSentinels=${BASH_REMATCH[1]}
        if [ "$masterIP" != "" ] && [ "$masterPort" == "6379" ]; then
            ((conditionsMet++))
            debug "\t* Master ${CHECK_SIGN}"
        else
            debug "\t* Master ${CROSS_SIGN}"    
        fi
        if [ "$numSlaves" == "2" ]; then
            ((conditionsMet++))
            debug "\t* Slaves ${CHECK_SIGN}"
        else
            debug "\t* Slaves ${CROSS_SIGN}"    
        fi
        if [ "$numOtherSentinels" == "2" ]; then
            ((conditionsMet++))
            debug "\t* Sentinels ${CHECK_SIGN}"
        else
            debug "\t* Sentinels ${CROSS_SIGN}"    
        fi
        if [ $conditionsMet == 3 ]; then
            break
        fi
        ((retryNo++))
        if [ $retryNo == $maxRetries ]; then
            fatal "Could not validate sentinel configuration after ${maxRetries} retries."
        fi
        sleep $retryNo
    done
}

# checkRedisInstance verifies that a Redis instance is up, by PINGing it.
# First argument is a string representing the Redis instance ip/hostname.
# Second argument is an integer representing the Redis instance port.
checkRedisInstance() {
    redisInstance=$1
    redisPort=$2
    retryNo=0
    maxRetries=5
    while true ; do
        reply=$(docker exec "$redisInstance" redis-cli -p "$redisPort" ping 2> /dev/null)
        if [ "$reply" == "PONG" ]; then
            debug "PING ${redisInstance}:${redisPort} ${CHECK_SIGN}"
            break
        else
            debug "PING ${redisInstance}:${redisPort} ${CROSS_SIGN}"    
        fi
        ((retryNo++))
        if [ $retryNo == $maxRetries ]; then
            fatal "Could not receive PING response from ${redisInstance}:${redisPort} after ${maxRetries} retries."
        fi
        sleep $retryNo
    done
}

### Main Execution
checkCommands "docker" "awk" "grep" "xargs"
debug "Running integration ${RUN}s in Redis '${REDIS_SETUP}' setup."
setUp
if [ "$REDIS_SETUP" == "single" ]; then
    setUpRedisSingleInstance
    runTests "xcache-redis6-single:6379" "" "xcache-redis7-single:6379" ""
elif [ "$REDIS_SETUP" == "failover" ]; then
    setUpRedisFailover
    redis6Sentinels="xcache-redis6-sentinel-1:26379,xcache-redis6-sentinel-2:26379,xcache-redis6-sentinel-3:26379"
    redis7Sentinels="xcache-redis7-sentinel-1:26379,xcache-redis7-sentinel-2:26379,xcache-redis7-sentinel-3:26379"
    runTests $redis6Sentinels "xcacheMaster" $redis7Sentinels "xcacheMaster"
else
    setUpRedisCluster
    redis6Nodes="127.0.0.1:6001,127.0.0.1:6002,127.0.0.1:6003"
    redis7Nodes="127.0.0.1:7001,127.0.0.1:7002,127.0.0.1:7003"
    runTests $redis6Nodes "" $redis7Nodes ""
fi
tearDown
debug "Done."
