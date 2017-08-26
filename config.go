package main

import (
	"bytes"
	"strconv"
	"strings"
	"unicode"
)

// config is the cgitweb config, loaded form either /usr/local/etc/cgitwebrc or the first argument
// of cgitweb.cgi.
var config = Config{}

type Config map[string][]string

func (cfg Config) toEnvironment() (env []string) {
	for k, vs := range cfg {
		for _, v := range vs {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func (cfg Config) getPrefix(prefix string, stripPrefix bool) Config {
	out := Config{}
	plen := len(prefix)
	for k, v := range cfg {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if stripPrefix {
			k = k[plen:]
		}
		out[k] = v
	}
	return out
}

func (cfg Config) getStrings(name string, defaultValues ...string) []string {
	strs := cfg[name]
	if len(strs) == 0 {
		return defaultValues
	}
	return strs
}

func (cfg Config) getInt64(name string, defaultValue int64) int64 {
	strs := cfg[name]
	for i := len(strs) - 1; i >= 0; i-- {
		num, err := strconv.ParseInt(strs[i], 0, 64)
		if err == nil {
			return num
		}
	}
	return defaultValue
}

func (cfg Config) getBool(name string, defaultValue bool) bool {
	strs := cfg[name]
	for i := len(strs) - 1; i >= 0; i-- {
		switch strings.ToLower(strs[i]) {
		case "t", "true", "yes", "on", "1":
			return true
		case "f", "false", "no", "off", "0":
			return false
		}
	}
	return defaultValue
}

func (cfg Config) getString(name string, allowEmpty bool, defaultValue string) string {
	strs := cfg[name]
	for i := len(strs) - 1; i >= 0; i-- {
		s := strs[i]
		if s == "" && !allowEmpty {
			continue
		}
		return s
	}
	return defaultValue
}

func (cfg Config) load(confData []byte) {
	for _, line := range bytes.Split(confData, []byte{'\n'}) {
		if len(line) == 0 || unicode.IsSpace(rune(line[0])) || line[0] == '#' {
			continue
		}
		var key, value string
		if eq := bytes.IndexByte(line, '='); eq > 0 {
			value, key = string(line[eq+1:]), string(line[:eq])
		} else {
			value, key = "TRUE", string(line)
		}

		if key == "" {
			continue
		}
		config[key] = append(config[key], value)
	}
}
