# encoding: UTF-8
require_relative 'out_elasticsearch'

class Fluent::ElasticsearchOutputDynamic < Fluent::ElasticsearchOutput

  Fluent::Plugin.register_output('elasticsearch_dynamic', self)

  config_param :delimiter, :string, :default => "."

  DYNAMIC_PARAM_NAMES = %W[hosts host port logstash_format logstash_prefix logstash_dateformat time_key utc_index index_name tag_key type_name id_key parent_key routing_key write_operation]
  DYNAMIC_PARAM_SYMBOLS = DYNAMIC_PARAM_NAMES.map { |n| "@#{n}".to_sym }

  attr_reader :dynamic_config

  def configure(conf)
    super

    # evaluate all configurations here
    @dynamic_config = {}
    DYNAMIC_PARAM_SYMBOLS.each_with_index { |var, i|
      value = expand_param(self.instance_variable_get(var), nil, nil, nil)
      key = DYNAMIC_PARAM_NAMES[i]
      @dynamic_config[key] = value.to_s
    }
    # end eval all configs
    @current_config = nil
  end

  def create_meta_config_map
    {'id_key' => '_id', 'parent_key' => '_parent', 'routing_key' => '_routing'}
  end

  def client(host)

    # check here to see if we already have a client connection for the given host
    connection_options = get_connection_options(host)

    @_es = nil unless is_existing_connection(connection_options[:hosts])

    @_es ||= begin
      @current_config = connection_options[:hosts].clone
      excon_options = { client_key: @client_key, client_cert: @client_cert, client_key_pass: @client_key_pass }
      adapter_conf = lambda {|f| f.adapter :excon, excon_options }
      transport = Elasticsearch::Transport::Transport::HTTP::Faraday.new(connection_options.merge(
                                                                          options: {
                                                                            reload_connections: @reload_connections,
                                                                            reload_on_failure: @reload_on_failure,
                                                                            resurrect_after: @resurrect_after,
                                                                            retry_on_failure: 5,
                                                                            transport_options: {
                                                                              request: { timeout: @request_timeout },
                                                                              ssl: { verify: @ssl_verify, ca_file: @ca_file }
                                                                            }
                                                                          }), &adapter_conf)
      es = Elasticsearch::Client.new transport: transport

      begin
        raise ConnectionFailure, "Can not reach Elasticsearch cluster (#{connection_options_description(host)})!" unless es.ping
      rescue *es.transport.host_unreachable_exceptions => e
        raise ConnectionFailure, "Can not reach Elasticsearch cluster (#{connection_options_description(host)})! #{e.message}"
      end

      log.info "Connection opened to Elasticsearch cluster => #{connection_options_description(host)}"
      es
    end
  end

  def get_connection_options(con_host)
    raise "`password` must be present if `user` is present" if @user && !@password

    hosts = if con_host || @hosts
      (con_host || @hosts).split(',').map do |host_str|
        # Support legacy hosts format host:port,host:port,host:port...
        if host_str.match(%r{^[^:]+(\:\d+)?$})
          {
            host:   host_str.split(':')[0],
            port:   (host_str.split(':')[1] || @port).to_i,
            scheme: @scheme
          }
        else
          # New hosts format expects URLs such as http://logs.foo.com,https://john:pass@logs2.foo.com/elastic
          uri = URI(host_str)
          %w(user password path).inject(host: uri.host, port: uri.port, scheme: uri.scheme) do |hash, key|
            hash[key.to_sym] = uri.public_send(key) unless uri.public_send(key).nil? || uri.public_send(key) == ''
            hash
          end
        end
      end.compact
    else
      [{host: @host, port: @port.to_i, scheme: @scheme}]
    end.each do |host|
      host.merge!(user: @user, password: @password) if !host[:user] && @user
      host.merge!(path: @path) if !host[:path] && @path
    end

    {
      hosts: hosts
    }
  end

  def connection_options_description(host)
    get_connection_options(host)[:hosts].map do |host_info|
      attributes = host_info.dup
      attributes[:password] = 'obfuscated' if attributes.has_key?(:password)
      attributes.inspect
    end.join(', ')
  end

  def write_objects(tag, chunk)
    bulk_message = Hash.new { |h,k| h[k] = '' }
    dynamic_conf = @dynamic_config.clone

    headers = {
      UPDATE_OP => {},
      UPSERT_OP => {},
      CREATE_OP => {},
      INDEX_OP => {}
    }

    chunk.msgpack_each do |time, record|
      next unless record.is_a? Hash

      # evaluate all configurations here
      DYNAMIC_PARAM_SYMBOLS.each_with_index { |var, i|
        k = DYNAMIC_PARAM_NAMES[i]
        v = self.instance_variable_get(var)
        # check here to determine if we should evaluate
        if dynamic_conf[k] != v
          value = expand_param(v, tag, time, record)
          dynamic_conf[k] = value
        end
      }
      # end eval all configs

      if eval_or_val(dynamic_conf['logstash_format'])
        if record.has_key?("@timestamp")
          time = Time.parse record["@timestamp"]
        elsif record.has_key?(dynamic_conf['time_key'])
          time = Time.parse record[dynamic_conf['time_key']]
          record['@timestamp'] = record[dynamic_conf['time_key']] unless time_key_exclude_timestamp
        else
          record.merge!({"@timestamp" => Time.at(time).to_datetime.to_s})
        end

        if eval_or_val(dynamic_conf['utc_index'])
          target_index = "#{dynamic_conf['logstash_prefix']}-#{Time.at(time).getutc.strftime("#{dynamic_conf['logstash_dateformat']}")}"
        else
          target_index = "#{dynamic_conf['logstash_prefix']}-#{Time.at(time).strftime("#{dynamic_conf['logstash_dateformat']}")}"
        end
      else
        target_index = dynamic_conf['index_name']
      end

      # Change target_index to lower-case since Elasticsearch doesn't
      # allow upper-case characters in index names.
      target_index = target_index.downcase

      if @include_tag_key
        record.merge!(dynamic_conf['tag_key'] => tag)
      end

      meta = {"_index" => target_index, "_type" => dynamic_conf['type_name']}

      @meta_config_map.each_pair do |config_name, meta_key|
        if dynamic_conf[config_name] && record[dynamic_conf[config_name]]
          meta[meta_key] = record[dynamic_conf[config_name]]
        end
      end

      if dynamic_conf['hosts']
        host = dynamic_conf['hosts']
      else
        host = "#{dynamic_conf['host']}:#{dynamic_conf['port']}"
      end

      if @remove_keys
        @remove_keys.each { |key| record.delete(key) }
      end

      write_op = dynamic_conf["write_operation"]
      append_record_to_messages(write_op, meta, headers[write_op], record, bulk_message[host])
    end

    bulk_message.each do |hKey, msgs|
      send_bulk(msgs, hKey) unless msgs.empty?
      msgs.clear
    end
  end

  def send_bulk(data, host)
    retries = 0
    begin
      client(host).bulk body: data
    rescue *client(host).transport.host_unreachable_exceptions => e
      if retries < 2
        retries += 1
        @_es = nil
        log.warn "Could not push logs to Elasticsearch, resetting connection and trying again. #{e.message}"
        sleep 2**retries
        retry
      end
      raise ConnectionFailure, "Could not push logs to Elasticsearch after #{retries} retries. #{e.message}"
    rescue Exception
      @_es = nil if @reconnect_on_error
      raise
    end
  end

  def eval_or_val(var)
    return var unless var.is_a?(String)
    eval(var)
  end

  def expand_param(param, tag, time, record)
    # check for '${ ... }'
    #   yes => `eval`
    #   no  => return param
    return param if (param =~ /\${.+}/).nil?

    # check for 'tag_parts[]'
      # separated by a delimiter (default '.')
    tag_parts = tag.split(@delimiter) unless (param =~ /tag_parts\[.+\]/).nil? || tag.nil?

    # pull out section between ${} then eval
    inner = param.clone
    while inner.match(/\${.+}/)
      to_eval = inner.match(/\${(.+?)}/){$1}

      if !(to_eval =~ /record\[.+\]/).nil? && record.nil?
        return to_eval
      elsif !(to_eval =~/tag_parts\[.+\]/).nil? && tag_parts.nil?
        return to_eval
      elsif !(to_eval =~/time/).nil? && time.nil?
        return to_eval
      else
        inner.sub!(/\${.+?}/, eval( to_eval ))
      end
    end
    inner
  end

  def is_valid_expand_param_type(param)
    return false if [:@buffer_type].include?(param)
    return self.instance_variable_get(param).is_a?(String)
  end

  def is_existing_connection(host)
    # check if the host provided match the current connection
    return false if @_es.nil?
    return false if host.length != @current_config.length

    for i in 0...host.length
      if !host[i][:host].eql? @current_config[i][:host] || host[i][:port] != @current_config[i][:port]
        return false
      end
    end

    return true
  end
end
