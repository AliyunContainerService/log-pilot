Build
=====

```
cd images
./build.sh
```

Run
===

```
docker run --rm -it --net host \
-v /var/run/docker.sock:/var/run/docker.sock \
-v /:/host \
-e FLUENTD_OUTPUT=elasticsearch \
-e ELASTICSEARCH_HOST=127.0.0.1 \
-e ELASTICSEARCH_PORT=9200 pilot
```

