#!/bin/bash

cd $(dirname $0)

green(){
    echo -e "\033[0;32m$*\033[0m"
}

blue(){
    echo -e "\033[0;34m$*\033[0m"
}

blue "Cleanup"
docker-compose -p quickstart -f es.yml down 

blue "Starting elasticsearch+kibana+fluentd-pilot"
docker-compose -p quickstart -f es.yml up -d

host=127.0.0.1
if [ -n "$DOCKER_HOST" ]; then
    host=$(echo $DOCKER_HOST|sed -e 's:^.*//::' -e 's/:.*$//')
fi
if [ -z "$host" ]; then
    echo "Could not detect docker host."
fi

time=0
while :
do
    echo -n -e "\rWaiting elasticsearch ready. ${time}s"
    http_code=$(curl -m 1 -s -o /dev/null -w '%{http_code}' http://$host:9200/)
    if ((http_code == 0)); then
        sleep 1
        ((time++))
        continue
    elif ((http_code == 200)); then
        break
    elif ((http_code >= 500)); then
        echo -e "\nstart fail"
        exit
    fi
done

blue "\nCleanup"
docker-compose -p tomcat -f tomcat.yml down 

blue "Starting tomcat"
docker-compose -p tomcat -f tomcat.yml up -d

pwd=$(pwd)
project=$(basename $pwd)


echo 
green "Start successfully!"
echo

cat << EOF
"Now open the urls below with your favorite broswer.
first access tomcat(generate some logs): http://$host:8080  
kibana(query the logs tomcat generated): http://$host:5601
Before you can see any logs,
you need to create new indexes "catalina" or "access".
EOF

