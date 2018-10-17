FROM golang:1.9-alpine3.6 as builder

ENV PILOT_DIR /go/src/github.com/AliyunContainerService/log-pilot
ARG GOOS=linux
ARG GOARCH=amd64
RUN set -ex && apk add --no-cache make git
WORKDIR $PILOT_DIR
COPY . $PILOT_DIR
RUN go install 

FROM alpine:3.6
COPY assets/glibc/glibc-2.26-r0.apk /tmp/
RUN apk update && \ 
    apk add python && \
    apk add ruby-json ruby-irb && \
    apk add build-base ruby-dev && \
    apk add python && \
    apk add lsof && \
    apk add ca-certificates wget && \
    gem install fluentd -v 1.2.6 --no-ri --no-rdoc && \
    gem install fluent-plugin-elasticsearch -v ">=2.0.0" --no-ri --no-rdoc && \
    gem install gelf -v "~> 3.0.0" --no-ri --no-rdoc && \
    gem install aliyun_sls_sdk -v ">=0.0.9" --no-ri --no-rdoc && \
    gem install remote_syslog_logger -v ">=1.0.1" --no-ri --no-rdoc && \
    gem install fluent-plugin-remote_syslog -v ">=0.2.1" --no-ri --no-rdoc && \
    gem install fluent-plugin-kafka --no-ri --no-rdoc && \
    gem install fluent-plugin-flowcounter --no-ri --no-rdoc && \
    apk del build-base ruby-dev && \
    rm -rf /root/.gem && \
    apk add curl openssl && \
    apk add --allow-untrusted /tmp/glibc-2.26-r0.apk && \
    rm -rf /tmp/glibc-2.26-r0.apk

COPY --from=builder /go/bin/log-pilot /pilot/pilot
COPY assets/entrypoint assets/fluentd/ assets/healthz /pilot/
RUN mkdir -p /etc/fluentd && \
    mv /pilot/plugins /etc/fluentd/ && \
    chmod +x /pilot/pilot /pilot/entrypoint /pilot/healthz /pilot/config.fluentd

HEALTHCHECK CMD /pilot/healthz

VOLUME /etc/fluentd/conf.d
VOLUME /pilot/pos
WORKDIR /pilot/
ENV PILOT_TYPE=fluentd
ENTRYPOINT ["/pilot/entrypoint"]
