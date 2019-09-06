Architecture
============

On every docker host, run a log-pilot instance. Log-pilot will monitor docker events, and parse log labels on new docker conatainer, and generate appropriate log configuration and notify fluentd or filebeat process to reload the new configuration.

![Architecture](architecture.png)

Run Log-pilot With Fluentd Plugin
=================================

You must set environment variable ```PILOT_TYPE=fluentd``` to enable fluentd plugin within log-pilot.

### Start log-pilot in docker container

```
docker run --rm -it \
   -v /var/run/docker.sock:/var/run/docker.sock \
   -v /etc/localtime:/etc/localtime \
   -v /:/host:ro \
   --cap-add SYS_ADMIN \
   registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9.5-fluentd
```

By default, all the logs that log-pilot collect will write to log-pilot's stdout. 

### Work with elastichsearch

The command below run pilot with elastichsearch output, this makes log-pilot send all logs to elastichsearch.

```
docker run --rm -it \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /:/host:ro \
    --cap-add SYS_ADMIN \
    -e LOGGING_OUTPUT=elasticsearch \
    -e ELASTICSEARCH_HOST=${ELASTICSEARCH_HOST} \
    -e ELASTICSEARCH_PORT=${ELASTICSEARCH_PORT} \
    registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9.5-fluentd
```

Log output plugin configuration
===============================

You can config the environment variable ```LOGGING_OUTPUT``` to determine which log management will be output. You can also config the environment variable ```FLUENTD_LOG_LEVEL``` to change the log level of fluentd, the default fluentd log level is ```info```.

### Supported log management

When using log-pilot with fluentd plugin to collect docker logs, you can config the following buffered environment variables:

```
FLUENTD_BUFFER_TYPE         "(optinal) buffer type: file|memory"
FLUENTD_FLUSH_INTERVAL      "(optinal) the interval in seconds to wait before invoking the next buffer flush, default is 60"
FLUENTD_FLUSH_MODE          "(optinal) flush mode: default|lazy|interval|immediate"
FLUENTD_RETRY_LIMIT         "(optinal) the limit on the number of retries before buffered data is discarded"
FLUENTD_RETRY_WAIT          "(optinal) seconds to wait before next retry to flush, default is 1"
```

Supported log output plugin:

- elasticsearch

```
ELASTICSEARCH_HOST       "(required) elasticsearch host"
ELASTICSEARCH_PORT       "(required) elasticsearch port"
ELASTICSEARCH_USER       "(optinal) elasticsearch authentication username"
ELASTICSEARCH_PASSWORD   "(optinal) elasticsearch authentication password"
ELASTICSEARCH_PATH       "(optinal) elasticsearch http path prefix"
ELASTICSEARCH_SCHEME     "(optinal) elasticsearch scheme, default is http"
ELASTICSEARCH_SSL_VERIFY "(optinal) need ssl verification, default is true"
```

- graylog

```
GRAYLOG_HOST             "(required) graylog host"
GRAYLOG_PORT             "(required) graylog port"
GRAYLOG_PROTOCOL         "(optinal) graylog protocol, default is udp"
```

not support buffered env

- aliyun_sls

```
ALIYUNSLS_PROJECT                     "(required) aliyun sls project"
ALIYUNSLS_REGION_ENDPOINT             "(required) aliyun sls region endpoint"
ALIYUNSLS_ACCESS_KEY_ID               "(required) aliyun access key"
ALIYUNSLS_ACCESS_KEY_SECRET           "(required) aliyun access secret"
SSL_VERIFY                            "(optinal) need ssl verification, default is false"
ALIYUNSLS_NEED_CREATE_LOGSTORE        "(optinal) create logstore if not exist, default is false"
ALIYUNSLS_CREATE_LOGSTORE_TTL         "(optinal) create logstore ttl, default is infinite"
ALIYUNSLS_CREATE_LOGSTORE_SHARD_COUNT "(optinal) create logstore shard count, default is 2"
```

- file

```
FILE_PATH      "(required) output log file directory"
FILE_COMPRESS  "(optinal) need compression, default is false"
FILE_FORMAT    "(optinal) output log format, default is json"
```

- syslog

```
SYSLOG_HOST    "(required) syslog host"
SYSLOG_PORT    "(required) syslog port"
```

- kafka

```
KAFKA_BROKERS                "(required) kafka brokers"
KAFKA_DEFAULT_TOPIC          "(optinal) default topic"
KAFKA_DEFAULT_PARTITION_KEY  "(optianl) default partition key"
KAFKA_DEFAULT_MESSAGE_KEY    "(optianl) default message key"
KAFKA_OUTPUT_DATA_TYPE       "(optianl) output data type: json|ltsv|msgpack"
KAFKA_MAX_SEND_RETRIES       "(optianl) the number of times to retry sending of messages to a leader, default is 1"
KAFKA_REQUIRED_ACKS          "(optianl) the number of acks required per request, default is -1"
KAFKA_ACK_TIMEOUT            "(optianl) how long the producer waits for acks, unit is seconds"
```

### Other log management

Log-pilot also support graylog2. Supports for other log managements are in progress. You are welcome to create a pull request.

Declare log configuration of docker container
=============================================

### Basic usage

```
docker run -it --rm  -p 10080:8080 \
    -v /usr/local/tomcat/logs \
    --label aliyun.logs.catalina=stdout \
    --label aliyun.logs.access=/usr/local/tomcat/logs/localhost_access_log.*.txt \
    tomcat
```

The command above runs tomcat container, expect that log-pilot collect stdout of tomcat and logs in `/usr/local/tomcat/logs/localhost_access_log.\*.txt`. `-v /usr/local/tomcat/logs` is needed here so fluentd-pilot could access file in tomcat container.

### More

There are many labels you can use to describe the log info. 

- `aliyun.logs.$name=$path`
    - Name is an identify, can be any string you want. The valid characters in name are `0-9a-zA-Z_-`
    - Path is the log file path, can contains wildcard. `stdout` is a special value which means stdout of the container.
- `aliyun.logs.$name.format=none|json|csv|nginx|apache2|regexp` format of the log
    - none: pure text.
    - json: a json object per line.
    - regexp: use regex parse log. The pattern is specified by `aliyun.logs.$name.format.pattern = $regex`
- `aliyun.logs.$name.tags="k1=v1,k2=v2"`: tags will be appended to log. 
- `aliyun.logs.$name.target=target-for-log-storage`: target is used by the output plugins, instruct the plugins to store
logs in appropriate place. For elasticsearch output, target means the log index in elasticsearch. For aliyun_sls output,
target means the logstore in aliyun sls. The default value of target is the log name.
