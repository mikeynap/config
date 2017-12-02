package config

import (
	"reflect"
	"strings"
)

func flagString(parent, field string) string {
	if len(parent) == 0 {
		return strings.ToLower(hyphen(field))
	}
	if len(field) == 0 {
		return strings.ToLower(hyphen(parent))
	}
	return strings.ToLower(hyphen(parent) + "-" + hyphen(field))
}

func envString(parent, field string) string {
	if len(parent) == 0 {
		return strings.ToUpper(underscore(field))
	}
	if len(field) == 0 {
		return strings.ToUpper(underscore(parent))
	}
	return strings.ToUpper(underscore(parent) + "_" + underscore(field))
}

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isLower(c byte) bool {
	return !isUpper(c)
}

func toUpper(c byte) byte {
	return c - ('a' - 'A')
}

func toLower(c byte) byte {
	return c + ('a' - 'A')
}

// Underscore converts "CamelCasedString" to "camel_cased_string".
func underscore(s string) string {
	return splitCamel(s, '_')
}

// Underscore converts "CamelCasedString" to "camel-cased-string".
func hyphen(s string) string {
	return splitCamel(s, '-')
}

func splitCamel(s string, sep byte) string {
	r := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUpper(c) {
			if i > 0 && i+1 < len(s) && (isLower(s[i-1]) || isLower(s[i+1])) {
				r = append(r, sep, toLower(c))
			} else {
				r = append(r, toLower(c))
			}
		} else {
			r = append(r, c)
		}
	}
	return string(r)
}

func contains(s []string, str string) bool {
	for _, si := range s {
		if si == str {
			return true
		}
	}
	return false
}

func isZero(x interface{}) bool {
	if v := reflect.ValueOf(x); v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
		return v.Len() == 0
	}
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

func isZeroStr(x string) bool {
	return x == "" || x == "0" || x == "0.0" || x == "[]"
}
