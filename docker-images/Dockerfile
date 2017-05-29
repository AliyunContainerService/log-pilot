FROM alpine:3.5

RUN apk update && apk upgrade && \  
  apk add ruby-json ruby-irb && \
  apk add build-base ruby-dev && \
  apk add ca-certificates wget && \
  gem install fluentd -v "~> 0.12.0" --no-ri --no-rdoc && \
  gem install fluent-plugin-elasticsearch --no-ri --no-rdoc && \
  gem install gelf -v "~> 3.0.0" --no-ri --no-rdoc && \
  gem install aliyun_sls_sdk -v ">=0.0.9" --no-ri --no-rdoc && \
  git clone git clone -b add_topic_key_config https://github.com/RussellCloud/fluent-plugin-kafka.git && \
  cd fluent-plugin-kafka/ && \
  gem build fluent-plugin-kafka.gemspec && \
  gem install fluent-plugin-kafka-0.5.5.gem && \
  cd ../ && \
  gem install zookeeper --no-ri --no-rdoc&& \
  gem install ruby-kafka --no-ri --no-rdoc&& \
  apk del build-base ruby-dev && \
  rm -rf /root/.gem

COPY plugins/ /etc/fluentd/plugins/

VOLUME /etc/fluentd/conf.d

COPY pilot fluentd.tpl entrypoint config.default /pilot/
VOLUME /pilot/pos

EXPOSE 24224
#CMD exec fluentd -c /fluentd/etc/$FLUENTD_CONF -p /fluentd/plugins $FLUENTD_OPT
CMD /pilot/entrypoint

