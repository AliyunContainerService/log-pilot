require 'fluent/filter'

module Fluent
  class DockerLogReserveFilter < Filter
    Fluent::Plugin.register_filter('contians_log_json_parser', self)

    # config_param works like other plugins
    config_param :jsonkey, :string, :default => 'log'

    def configure(conf)
      super
      # do the usual configuration here
    end

    def start
      super
      # This is the first method to be called when it starts running
      # Use it to allocate resources, etc.
    end

    def shutdown
      super
      # This method is called when Fluentd is shutting down.
      # Use it to free up resources, etc.
    end

    def filter(tag, time, record)
      if record.nil?
        return
      end
      if @jsonkey.nil? or !record.has_key?(@jsonkey)
        return record
      end

      begin
        h = JSON.parse(record[@jsonkey])
        h.each do |k,v|
          # adapt sls fmt
          out = (v.is_a?(Hash)) ? ("#{v}") : (v)
          out = (v.is_a?(Numeric)) ? ("#{v}") : (v)
          record[k] = out
        end
        record.delete(@jsonkey)
      rescue
      end
      return record
    end
  end
end