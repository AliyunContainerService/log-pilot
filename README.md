fluentd-pilot
=============

`fluentd-pilot` is an awesome docker log tool. With `fluentd-pilot` you can collect logs from docker hosts and send them to your centralize log system such as elastichsearch, graylog2, awsog and etc. `fluentd-pilot` can collect not only docker stdout but also log file that inside docker containers.

Quickstart
==========

### Run pilot

```
docker run --rm -it \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /:/host \
    registry.cn-hangzhou.aliyuncs.com/acs-sample/fluentd-pilot:latest
```

### Run applications whose logs need to be collected

Open a new terminal, run the application. With tomcat for example:

```
docker run -it --rm  -p 10080:8080 \
    -v /usr/local/tomcat/logs \
    --label aliyun.logs.catalina=stdout \
    --label aliyun.logs.access=/usr/local/tomcat/logs/localhost_access_log.*.txt \
    tomcat
```

Now watch the output of fluentd-pilot. You will find that fluentd-pilot get all tomcat's startup logs. If you access tomcat with your broswer, access logs in `/usr/local/tomcat/logs/localhost_access_log.\*.txt` will also be displayed in fluentd-pilot's output.

Feature
========

- Single fluentd process per docker host. You don't need to create new fluentd process for every docker container.
- Support both stdout and file. Either Docker log driver or logspout can only collect stdout.
- Declarative configuration. You need do nothing but declare the logs you want to collect.
- Support many log management: elastichsearch, graylog2, awslogs and more.
- Tags. You could add tags on the logs collected, and later filter by tags in log management.

Contribute
==========

You are welcome to make new issues and pull reuqests.

