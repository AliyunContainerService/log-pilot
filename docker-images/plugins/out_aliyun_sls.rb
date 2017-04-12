module Fluent

  class AliyunSlsOutput < BufferedOutput
    Plugin.register_output('aliyun_sls', self)

    config_param :project, :string, :default => nil
    config_param :region_endpoint, :string, :default => nil
    config_param :access_key_id, :string, :default => nil
    config_param :access_key_secret, :string, :default => nil
    config_param :ssl_verify, :bool, :default => false
    config_param :need_create_logstore, :bool, :default => false
    config_param :create_logstore_ttl, :integer, :default => 1
    config_param :create_logstore_shard_count, :integer, :default => 2

    def initialize
      super
      require "aliyun_sls_sdk/protobuf"
      require "aliyun_sls_sdk"
      @log_store_created = false
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
      if record["@target"]
        [tag, time, record].to_msgpack
      else
        super
      end
    end

    def client
      @topic = `hostname`.strip
      @_sls_con ||= AliyunSlsSdk::LogClient.new(@region_endpoint, @access_key_id, @access_key_secret, @ssl_verify)
    end

    def createLogStore(logstore_name)
      begin
        getStoreResp = client.get_logstore(@project, logstore_name)
      rescue AliyunSlsSdk::LogException => e
        if e.errorCode == "LogStoreNotExist"
          retries = 2
          begin
            createLogStoreResp = logClient.create_logstore(@project, logstore_name, @create_logstore_ttl, @create_logstore_shard_count)
          rescue AliyunSlsSdk::LogException => e
            if e.errorCode == "LogstoreAlreadyExist"
              log.warn "logstore #{logstore_name} already exist"
            else
              raise
            end
          rescue => e
            if retries > 0
              log.warn "Error caught when creating logs store: #{e}"
              retries -= 1
              retry
            end
          end
        end
      end
    end

    def getLogItem(record)
      contents = {}
      record.each { |k, v|
        contents[k] = v
      }
      AliyunSlsSdk::LogItem.new(nil, contents)
    end

    def write(chunk)
      log_list_hash = {}
      chunk.msgpack_each do |tag, time, record|
        if record and record["@target"]
          logStoreName = record["@target"]
          if not @log_store_created
            if @need_create_logstore
              createLogStore(logStoreName)
              @log_store_created = true
            else
              @log_store_created = true
            end
          end
          record.delete("@target")
          if not log_list_hash[logStoreName]
            logItems = []
            log_list_hash[logStoreName] = logItems
          end
          log_list_hash[logStoreName] << getLogItem(record)
        else
          log.warn "no @target key in record: #{record}, tag: #{tag}, time: #{time}"
        end
      end

      puts log_list_hash.to_s
      log_list_hash.each do |storeName, logitems|
        puts storeName
        puts logitems
        putLogRequest = AliyunSlsSdk::PutLogsRequest.new(@project, storeName, @topic, nil, logitems, nil, true)
        retries = 2
        begin
          client.put_logs(putLogRequest)
        rescue => e
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
  end
end
