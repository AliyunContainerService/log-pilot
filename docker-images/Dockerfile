FROM alpine:3.5

ENV FILEBEAT_VERSION=6.1.1

COPY glibc/glibc-2.26-r0.apk /tmp/

RUN apk update && apk upgrade && \  
  apk add ruby-json ruby-irb && \
  apk add build-base ruby-dev && \
  apk add python && \
  apk add lsof && \
  apk add ca-certificates wget && \
  gem install fluentd -v "~> 0.12.0" --no-ri --no-rdoc && \
  gem install fluent-plugin-elasticsearch --no-ri --no-rdoc && \
  gem install gelf -v "~> 3.0.0" --no-ri --no-rdoc && \
  gem install aliyun_sls_sdk -v ">=0.0.9" --no-ri --no-rdoc && \
  gem install remote_syslog_logger -v ">=1.0.1" --no-ri --no-rdoc && \
  gem install fluent-plugin-remote_syslog -v ">=0.2.1" --no-ri --no-rdoc && \
  gem install fluent-plugin-kafka --no-ri --no-rdoc && \
  apk del build-base ruby-dev && \
  rm -rf /root/.gem && \
  apk add curl openssl && \
  apk add --allow-untrusted /tmp/glibc-2.26-r0.apk && \
  update-ca-certificates && \
  wget http://acs-logging.oss-cn-hangzhou.aliyuncs.com/beats/filebeat/filebeat-${FILEBEAT_VERSION}-linux-x86_64.tar.gz -P /tmp/ && \
  mkdir -p /usr/share/filebeat /usr/share/filebeat/data /usr/share/filebeat/logs /usr/share/filebeat/prospectors.d && \
  tar zxf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64.tar.gz -C /tmp/ && \
  cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/filebeat /usr/share/filebeat/ && \
  cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/fields.yml /usr/share/filebeat/ && \
  cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/kibana /usr/share/filebeat/ && \
  cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/module /usr/share/filebeat/ && \
  cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/modules.d /usr/share/filebeat/ && \
  rm -rf /var/cache/apk/* /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64.tar.gz /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64 && \
  rm -rf /tmp/glibc-2.26-r0.apk

COPY plugins/ /etc/fluentd/plugins/
COPY pilot fluentd.tpl filebeat.tpl entrypoint config.fluentd config.filebeat /pilot/

VOLUME /etc/fluentd/conf.d
VOLUME /pilot/pos

EXPOSE 24224

WORKDIR /pilot/

CMD /pilot/entrypoint
