package pilot

import (
	"io/ioutil"
	"strings"
)

// ReadFile return string list separated by separator
func ReadFile(path string, separator string) ([]string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(data), separator), nil
}
