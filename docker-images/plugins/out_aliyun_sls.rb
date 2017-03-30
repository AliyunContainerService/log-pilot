module Fluent

  class AliyunSlsOutput < BufferedOutput
    Plugin.register_output('aliyun_sls', self)

    config_param :project, :string, :default => nil
    config_param :region_endpoint, :string, :default => nil
    config_param :access_key_id, :string, :default => nil
    config_param :access_key_secret, :string, :default => nil
    config_param :ssl_verify, :bool, :default => false

    def initialize
      super
      require "aliyun_sls_sdk/protobuf"
      require "aliyun_sls_sdk/connection"
    end

    def configure(conf)
      super
    end


    def start
      super
    end

    def shutdown
      super
    end

    def format(tag, time, record)
      if record["target"]
        [tag, time, record].to_msgpack
      else
        super
      end
    end

    def client
      @topic = `hostname`.strip
      @_sls_con ||= AliyunSlsSdk::Connection.new(@project, @region_endpoint, @access_key_id, @access_key_secret)
    end

    def write(chunk)
      log_list_hash = {}
      chunk.msgpack_each do |tag, time, record|
        if record and record["target"]
          logStoreName = record["target"]
          if not log_list_hash[logStoreName]
            log_list = AliyunSlsSdk::Protobuf::LogGroup.new(:logs => [], :topic => @topic, :source => @source)
            log_list_hash[logStoreName] = log_list
          end
          log = AliyunSlsSdk::Protobuf::Log.new(:time => Time.now.to_i, :contents => [])
          pack_log_item(log_list_hash[logStoreName], log, record)
        else
          log.warn "no target key in record: #{record}, tag: #{tag}, time: #{time}"
        end
      end
      log_list_hash.each do |storeName, log_list|
        retries = 2
        begin
          client.puts_logs(storeName, log_list, @ssl_verify)
        rescue Exception => e
          if retries > 0
            log.warn "\tCaught in puts logs: #{e}"
            client.http.shutdown
            @_sls_con = nil
            retries -= 1
            retry
          end
          log.error "Could not puts logs to aliyun sls: #{e}"
        end
      end
    end

    private

    def pack_log_item(log_list, log, record)
      pack_hash_log_item(log, record)
      log_list.logs << log
    end

    def pack_hash_log_item(log, hash)
      if hash
        hash.each { |k, v|
          log_item = AliyunSlsSdk::Protobuf::Log::Content.new(:key => k, :value => v || "")
          log.contents << log_item
        }
      end
    end
  end
end
