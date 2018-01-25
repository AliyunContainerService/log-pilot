require 'fluent/filter'

class Fluent::AddTimeFilter < Fluent::Filter
  Fluent::Plugin.register_filter('add_time', self)

  config_param :time_key, :string, :default => 'time'
  config_param :time_format, :string, :default => '%Y-%m-%dT%H:%M:%S.%L'

  def initialize
    super
  end

  def configure(conf)
    super
  end

  def filter(tag, time, record)
	if record.nil? 
		return
	end
    record[@time_key] = Time.now.strftime @time_format
    return record
  end
end if defined?(Fluent::Filter)
