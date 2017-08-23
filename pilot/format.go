package pilot

import "fmt"

type FormatConverter func(info *LogInfoNode) (map[string]string, error)

var converters = make(map[string]FormatConverter)

func Register(format string, converter FormatConverter) {
	converters[format] = converter
}

func Convert(info *LogInfoNode) (map[string]string, error) {
	converter := converters[info.value]
	if converter == nil {
		return nil, fmt.Errorf("unsupported log format: %s", info.value)
	}
	return converter(info)
}

type SimpleConverter struct {
	properties map[string]bool
}

func init() {

	simpleConverter := func(properties []string) FormatConverter {
		return func(info *LogInfoNode) (map[string]string, error) {
			validProperties := make(map[string]bool)
			for _, property := range properties {
				validProperties[property] = true
			}
			ret := make(map[string]string)
			for k, v := range info.children {
				if _, ok := validProperties[k]; !ok {
					return nil, fmt.Errorf("%s is not a valid properties for format %s", k, info.value)
				}
				ret[k] = v.value
			}
			return ret, nil
		}
	}

	Register("none", simpleConverter([]string{}))
	Register("csv", simpleConverter([]string{"time_key", "time_format", "keys"}))
	Register("json", simpleConverter([]string{"time_key", "time_format"}))
	Register("regexp", simpleConverter([]string{"time_key", "time_format"}))
	Register("apache2", simpleConverter([]string{}))
	Register("apache_error", simpleConverter([]string{}))
	Register("nginx", simpleConverter([]string{}))
	Register("regexp", func(info *LogInfoNode) (map[string]string, error) {
		ret, err := simpleConverter([]string{"pattern", "time_format", "time_key", "keep_time_key"})(info)
		if err != nil {
			return ret, err
		}
		if ret["pattern"] == "" {
			return nil, fmt.Errorf("regex pattern can not be emtpy")
		}
		return ret, nil
	})
}
