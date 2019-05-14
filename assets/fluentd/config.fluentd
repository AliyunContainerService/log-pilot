#!/bin/sh

set -e
cd $(dirname $0)

FLUENTD_CONFIG=/etc/fluentd/fluentd.conf

assert_not_empty(){
    arg=$1
    shift
    if [ -z "$arg" ]; then
        echo "$@"
        exit 1
    fi
}

if [ -f "$FLUENTD_CONFIG" ]; then
    exit
fi

mkdir -p /etc/fluentd
echo "@include conf.d/*.conf" >> $FLUENTD_CONFIG


bufferd_output(){
cat << EOF
${FLUENTD_BUFFER_TYPE:+buffer_type $FLUENTD_BUFFER_TYPE}
${FLUENTD_BUFFER_CHUNK_LIMIT:+buffer_chunk_limit $FLUENTD_BUFFER_CHUNK_LIMIT}
${FLUENTD_BUFFER_QUEUE_LIMIT:+buffer_queue_limit $FLUENTD_BUFFER_QUEUE_LIMIT}
${FLUENTD_BUFFER_CHUNK_LIMIT_SIZE:+chunk_limit_size ${FLUENTD_BUFFER_CHUNK_LIMIT_SIZE}}
${FLUENTD_BUFFER_TOTAL_LIMIT_SIZE:+total_limit_size ${FLUENTD_BUFFER_TOTAL_LIMIT_SIZE}}
${FLUENTD_BUFFER_CHUNK_FULL_THRESHOLD:+chunk_full_threshold ${FLUENTD_BUFFER_CHUNK_FULL_THRESHOLD}}
${FLUENTD_BUFFER_COMPRESS:+compress ${FLUENTD_BUFFER_COMPRESS}}
${FLUENTD_FLUSH_INTERVAL:+flush_interval $FLUENTD_FLUSH_INTERVAL}
${FLUENTD_FLUSH_MODE:+flush_mode ${FLUENTD_FLUSH_MODE}}
${FLUENTD_FLUSH_THREAD_COUNT:+flush_thread_count ${FLUENTD_FLUSH_THREAD_COUNT}}
${FLUENTD_FLUSH_AT_SHUTDOWN:+flush_at_shutdown $FLUENTD_FLUSH_AT_SHUTDOWN}
${FLUENTD_DISABLE_RETRY_LIMIT:+disable_retry_limit $FLUENTD_DISABLE_RETRY_LIMIT}
${FLUENTD_RETRY_LIMIT:+retry_limit $FLUENTD_RETRY_LIMIT}
${FLUENTD_RETRY_WAIT:+retry_wait $FLUENTD_RETRY_WAIT}
${FLUENTD_MAX_RETRY_WAIT:+max_retry_wait $FLUENTD_MAX_RETRY_WAIT}
${FLUENTD_NUM_THREADS:+num_threads $FLUENTD_NUM_THREADS}
EOF
}

fluentd_options(){
cat >> $FLUENTD_CONFIG << EOF
<system>
${FLUENTD_LOG_LEVEL:+log_level $FLUENTD_LOG_LEVEL}
</system>
EOF
if [ "$FLUENTD_ENABLE_MONITOR" == "true" ]; then
cat >> $FLUENTD_CONFIG << EOF
<source>
    @type monitor_agent
    bind 0.0.0.0
    port 24220
</source>
EOF
fi
}

es(){
if [ -f "/run/secrets/es_credential" ];then
    ELASTICSEARCH_USER=$(cat /run/secrets/es_credential | awk -F":" '{ print $1 }')
    ELASTICSEARCH_PASSWORD=$(cat /run/secrets/es_credential | awk -F":" '{ print $2 }')
fi

if [ -z "$ELASTICSEARCH_HOSTS" ]; then
    assert_not_empty "$ELASTICSEARCH_HOST" "ELASTICSEARCH_HOST required"
    assert_not_empty "$ELASTICSEARCH_PORT" "ELASTICSEARCH_PORT required"
    ELASTICSEARCH_HOSTS="$ELASTICSEARCH_HOST:$ELASTICSEARCH_PORT"
fi

cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type elasticsearch
hosts $ELASTICSEARCH_HOSTS
reconnect_on_error true
${ELASTICSEARCH_USER:+user ${ELASTICSEARCH_USER}}
${ELASTICSEARCH_PASSWORD:+password ${ELASTICSEARCH_PASSWORD}}
${ELASTICSEARCH_PATH:+path ${ELASTICSEARCH_PATH}}
${ELASTICSEARCH_SCHEME:+scheme ${ELASTICSEARCH_SCHEME}}
${ELASTICSEARCH_SSL_VERIFY:+ssl_verify ${ELASTICSEARCH_SSL_VERIFY}}
target_index_key _target
type_name fluentd
$(bufferd_output)
</match>
EOF
}

default(){
echo "use default output"
cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type stdout
</match>
EOF
}

file(){
assert_not_empty "$FILE_PATH" "FILE_PATH required"
cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type file
path $FILE_PATH/\${docker_app}/\${docker_service}/\${docker_container}/\${tag[2]}.%Y-%m-%d
append ${FILE_APPEND:=true}
${FILE_COMPRESS:+compress ${FILE_COMPRESS}}
<format>
  @type ${FILE_FORMAT:=json}
</format>
<buffer tag,time,docker_app,docker_service,docker_container>
  @type ${FILE_BUFFER_TYPE:=file}
  path $FILE_PATH/.buffer
  timekey ${FILE_BUFFER_TIME_KEY:=1d}
  timekey_wait ${FILE_BUFFER_TIME_KEY_WAIT:=5m}
  timekey_use_utc ${FILE_BUFFER_TIME_KEY_USE_UTC:=false}
  $(bufferd_output)
</buffer>
</match>
EOF
}

graylog(){
assert_not_empty "$GRAYLOG_HOST" "GRAYLOG_HOST required"
assert_not_empty "$GRAYLOG_PORT" "GRAYLOG_PORT required"
cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type gelf
host $GRAYLOG_HOST
port $GRAYLOG_PORT
protocol ${GRAYLOG_PROTOCOL:-udp}
flush_interval 3s
$(bufferd_output)
</match>
EOF
}

aliyun_sls(){
if [ -f "/run/secrets/aliyun_access_key" ];then
    ALIYUNSLS_ACCESS_KEY_ID=$(cat /run/secrets/aliyun_access_key | awk -F":" '{ print $1 }')
    ALIYUNSLS_ACCESS_KEY_SECRET=$(cat /run/secrets/aliyun_access_key | awk -F":" '{ print $2 }')
fi

assert_not_empty "$ALIYUNSLS_PROJECT"         "ALIYUNSLS_PROJECT required"
assert_not_empty "$ALIYUNSLS_REGION_ENDPOINT" "ALIYUNSLS_REGION_ENDPOINT required"
assert_not_empty "$ALIYUNSLS_ACCESS_KEY_ID"   "ALIYUNSLS_ACCESS_KEY_ID required"
assert_not_empty "$ALIYUNSLS_ACCESS_KEY_SECRET"   "ALIYUNSLS_ACCESS_KEY_SECRET required"

cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type aliyun_sls

project              $ALIYUNSLS_PROJECT
region_endpoint      $ALIYUNSLS_REGION_ENDPOINT
access_key_id        $ALIYUNSLS_ACCESS_KEY_ID
access_key_secret    $ALIYUNSLS_ACCESS_KEY_SECRET
ssl_verify           ${SSL_VERIFY:-false}
need_create_logstore ${ALIYUNSLS_NEED_CREATE_LOGSTORE:-false}
create_logstore_ttl  ${ALIYUNSLS_CREATE_LOGSTORE_TTL:-1}
create_logstore_shard_count ${ALIYUNSLS_CREATE_LOGSTORE_SHARD_COUNT:-2}
$(bufferd_output)
</match>
EOF
}

syslog(){
assert_not_empty "$SYSLOG_HOST" "SYSLOG_HOST required"
assert_not_empty "$SYSLOG_PORT" "SYSLOG_PORT required"

cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type remote_syslog
host $SYSLOG_HOST
port $SYSLOG_PORT
${SYSLOG_FACILITY:+facility ${SYSLOG_FACILITY}}
${SYSLOG_SEVERITY:+facility ${SYSLOG_SEVERITY}}
tag ${SYSLOG_TAG:-fluentd-pilot}
</match>
EOF
}

kafka(){
assert_not_empty "$KAFKA_BROKERS" "KAFKA_BROKERS required"
cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type kafka_buffered
brokers $KAFKA_BROKERS
${KAFKA_DEFAULT_TOPIC:+default_topic $KAFKA_DEFAULT_TOPIC}
${KAFKA_DEFAULT_PARTITION_KEY:+default_partition_key $KAFKA_default_partition_key}
${KAFKA_DEFAULT_MESSAGE_KEY:+default_message_key $KAFKA_default_message_key}
${KAFKA_OUTPUT_DATA_TYPE:+output_data_type $KAFKA_OUTPUT_DATA_TYPE}
${KAFKA_OUTPUT_INCLUDE_TAG:+output_include_tag $KAFKA_OUTPUT_INCLUDE_TAG}
${KAFKA_OUTPUT_INCLUDE_TIME:+output_include_time $KAFKA_OUTPUT_INCLUDE_TIME}
${KAFKA_EXCLUDE_TOPIC_KEY:+exclude_topic_key $KAFKA_EXCLUDE_TOPIC_KEY}
${KAFKA_EXCLUDE_PARTITION_KEY:+exclude_partition_key $KAFKA_EXCLUDE_PARTITION_KEY}
${KAFKA_GET_KAFKA_CLIENT_LOG:+get_kafka_client_log $KAFKA_GET_KAFKA_CLIENT_LOG}
${KAFKA_MAX_SEND_RETRIES:+max_send_retries $KAFKA_MAX_SEND_RETRIES}
${KAFKA_REQUIRED_ACKS:+required_acks $KAFKA_REQUIRED_ACKS}
${KAFKA_ACK_TIMEOUT:+ack_timeout $KAFKA_ACK_TIMEOUT}
${KAFKA_COMPRESSION_CODEC:+compression_codec $KAFKA_COMPRESSION_CODEC}
${KAFKA_KAFKA_AGG_MAX_BYTES:+kafka_agg_max_bytes $KAFKA_KAFKA_AGG_MAX_BYTES}
${KAFKA_KAFKA_AGG_MAX_MESSAGES:+kafka_agg_max_messages $KAFKA_KAFKA_AGG_MAX_MESSAGES}
${KAFKA_MAX_SEND_LIMIT_BYTES:+max_send_limit_bytes $KAFKA_MAX_SEND_LIMIT_BYTES}
${KAFKA_DISCARD_KAFKA_DELIVERY_FAILED:+discard_kafka_delivery_failed $KAFKA_DISCARD_KAFKA_DELIVERY_FAILED}
$(bufferd_output)
</match>
EOF
}

null(){
cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type null
</match>
EOF
}

flowcounter() {
cat >> $FLUENTD_CONFIG << EOF
<match docker.**>
@type flowcounter
tag flowcounter
count_interval 30s
aggregate all
</match>
<match flowcounter>
@type stdout
</match>
EOF
}

if [ -n "$FLUENTD_OUTPUT" ]; then
    LOGGING_OUTPUT="$FLUENTD_OUTPUT"
fi

case "$LOGGING_OUTPUT" in
    elasticsearch)
        es;;
    graylog)
        graylog;;
    aliyun_sls)
        aliyun_sls;;
    file)
        file;;
    syslog)
        syslog;;
    kafka)
        kafka;;
    null)
        null;;
    flowcounter)
        flowcounter;;
    *)
        default
esac

fluentd_options
