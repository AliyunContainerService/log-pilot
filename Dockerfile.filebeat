FROM golang:1.9-alpine3.6 as builder

ENV PILOT_DIR /go/src/github.com/AliyunContainerService/log-pilot
ARG GOOS=linux
ARG GOARCH=amd64
RUN set -ex \
    && apk add --no-cache make git
WORKDIR $PILOT_DIR
COPY . $PILOT_DIR
RUN go install 

FROM alpine:3.6

ENV FILEBEAT_VERSION=6.1.1-2
COPY assets/glibc/glibc-2.26-r0.apk /tmp/
RUN apk update && \ 
    apk add python && \
    apk add ca-certificates && \
    apk add wget && \
    update-ca-certificates && \
    wget http://acs-logging.oss-cn-hangzhou.aliyuncs.com/beats/filebeat/filebeat-${FILEBEAT_VERSION}-linux-x86_64.tar.gz -P /tmp/ && \
    mkdir -p /etc/filebeat /var/lib/filebeat /var/log/filebeat && \
    tar zxf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64.tar.gz -C /tmp/ && \
    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/filebeat /usr/bin/ && \
    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/fields.yml /etc/filebeat/ && \
    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/kibana /etc/filebeat/ && \
    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/module /etc/filebeat/ && \
    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/modules.d /etc/filebeat/ && \
    apk add --allow-untrusted /tmp/glibc-2.26-r0.apk && \
    rm -rf /var/cache/apk/* /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64.tar.gz /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64 /tmp/glibc-2.26-r0.apk

COPY --from=builder /go/bin/log-pilot /pilot/pilot
COPY assets/entrypoint assets/filebeat/ /pilot/
VOLUME /var/log/filebeat
VOLUME /var/lib/filebeat

WORKDIR /pilot/
ENV PILOT_TYPE=filebeat
ENTRYPOINT ["/pilot/entrypoint"]
