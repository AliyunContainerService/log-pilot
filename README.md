fluentd-pilot
=============

fluentd-pilot是一个自动Docker容器日志的工具。只需要在机器上部署一个fluentd-pilot实例，就可以收集本机所有Docker容器日志。

QuickStart
==========

启动pilot

```
docker run --rm -it \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /:/host \
    registry.cn-hangzhou.aliyuncs.com/acs-sample/fluentd-pilot:latest
```

新开一个终端，运行一个要收集日志的应用，比如tomcat

```
docker run -it --rm  -p 10080:8080 \
    -v /usr/local/tomcat/logs \
    --label aliyun.logs.catalina=stdout \
    --label aliyun.logs.access=/usr/local/tomcat/logs/localhost_access_log.*.txt \
    tomcat
```

观察pilot的输出，可以看到tomcat所有的日志。

特性
====

- 一个单独fluentd进程，收集机器上所有容器的日志。不需要为每个容器启动一个fluentd进程
- 支持文件日志和stdout。docker log dirver亦或logspout只能处理stdout，fluentd-pilot不光支持收集stdout日志，还可以收集文件日志。
- 声明式配置。当你的容器有日志要收集，只要通过label声明要收集的日志文件的路径，无需改动其他任何配置，fluentd-pilot就会自动收集新容器的日志。
- 支持多种日志存储方式。无论是强大的阿里云日志服务，还是比较流行的elasticsearch组合，甚至是graylog，fluentd-pilot都能把日志投递到正确的地点。
- 支持tag。你可以在容器配置上增加一些tag，以便于过滤此容器的日志

参与开发
========

欢迎提issue和pr
