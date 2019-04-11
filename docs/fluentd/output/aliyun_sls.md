### aliyun sls output plugin

#### Environment variables for image `fluentd`

- LOGGING_OUTPUT=aliyun_sls  # Required, specify your output plugin name
- ALIYUNSLS_PROJECT=test-fluentd  # Required, specify your aliyun sls project name
- ALIYUNSLS_REGION_ENDPOINT=cn-hangzhou.log.aliyuncs.com  # Required, specify your region root endpoint
- ALIYUNSLS_ACCESS_KEY_ID="your aliyun access key id"     # Required 
- ALIYUNSLS_ACCESS_KEY_SECRET="your aliyun access key secret"  # Required 
- SSL_VERIFY="true" # Optional, use `https` scheme to access aliyun sls service, default is false.
- ALIYUNSLS_NEED_CREATE_LOGSTORE="true" # Optional, when set `true`, logstore will be created if not exist, default is false.
- ALIYUNSLS_CREATE_LOGSTORE_TTL=2  # Optional, used when ALIYUNSLS_NEED_CREATE_LOGSTORE set `true` and as param when creating log store, set the logging data time to live in days default is 1 day to live
- ALIYUNSLS_CREATE_LOGSTORE_SHARD_COUNT=2 # Optional, used when ALIYUNSLS_NEED_CREATE_LOGSTORE set `true` and as param when creating log store, set the shard count, default is 2 shard count 

#### Labels for your `app image` whose log will streamed to aliyun sls

- `aliyun.logs.${name}=${path}`
    - If there is an application name for your container, the aliyun sls `logstore name` will be `${container-app-name}-${name}`, or just `${name}` without application name. You should create the logstore first in aliyun sls console.

#### Save your access key in `docker secrets` in swarm mode
If you want to save the `aliyun access key` in docker secrets, you don't need to specify it with environment variables, instead:
````
# suppose "your_access_key_id" is your access_key_id
# suppose "your_access_key_secret" is your access_key_secret
$ echo "your_access_key_id:your_access_key_secret" | docker secret create aliyun_access_key - # create a secret named `aliyun_access_key`, the name is relevant and you should not change it
$ docker secret ls
ID                          NAME                CREATED             UPDATED
swm2ft9bzaxyyi9umwbb0mdd6   aliyun_access_key   53 minutes ago      53 minutes ago

````

#### Example for non-swarm-mode setup
* For `registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9.5-fluentd`
````
docker run --rm -it \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /etc/localtime:/etc/localtime \
    -v /:/host:ro \
    --cap-add SYS_ADMIN \
    -e LOGGING_OUTPUT=aliyun_sls \
    -e ALIYUNSLS_PROJECT="your-aliyun-sls-project-name"  \
    -e ALIYUNSLS_REGION_ENDPOINT=cn-hangzhou.log.aliyuncs.com \
    -e ALIYUNSLS_ACCESS_KEY_ID="your-access-key-id" \
    -e ALIYUNSLS_ACCESS_KEY_SECRET="your-access-key-secret"  \
    -e ALIYUNSLS_NEED_CREATE_LOGSTORE="true" \
    registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9.5-fluentd
````


* For your app, suppose a tomcat
````
docker run -it --rm  -p 10080:8080 \
    -v /usr/local/tomcat/logs \
    --label aliyun.logs.store-stdout=stdout \
    --label aliyun.logs.store-access=/usr/local/tomcat/logs/localhost_access_log.*.txt \
    tomcat
````
since there is no project name for `tomcat`, the stdout of `tomcat` will streamed to aliyun sls logstore `store-stdout` and the access log of `tomcat` will streamed to aliyun sls logstore `store-access`


#### Example for swarm-mode setup with docker secrets
* For `registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9.5-fluentd`
````
$ echo "your-access-key-id:your-access-key-secret" | docker secret create aliyun_access_key -
$ docker service create    -t \
    --mount type=bind,source=/var/run/docker.sock,destination=/var/run/docker.sock \
    --mount type=bind,source=/,destination=/host \
    --secret="aliyun_access_key" \
    -e LOGGING_OUTPUT=aliyun_sls \
    -e ALIYUNSLS_PROJECT="your-aliyun-sls-project-name"  \
    -e ALIYUNSLS_REGION_ENDPOINT=cn-hangzhou.log.aliyuncs.com \
    registry.cn-hangzhou.aliyuncs.com/acs/log-pilot:0.9.5-fluentd
````
* For your app, suppose a tomcat
````
docker run -it --rm  -p 10080:8080 \
    -v /usr/local/tomcat/logs \
    --label aliyun.logs.store-stdout=stdout \
    --label aliyun.logs.store-access=/usr/local/tomcat/logs/localhost_access_log.*.txt \
    tomcat
````

