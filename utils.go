package config

import (
	"fmt"
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

func sameCase(a, b byte) bool {
	return (a >= 'a' && b >= 'a') || (a <= 'Z' && b <= 'Z')
}

/*
"Test":         "test",
"TestThing":    "test-thing",
"TestThingT":   "test-thing-t",
"TestURLStuff": "test-url-stuff",
"TestURL":      "test-url",
"TestURLs":     "test-urls",
"TestUrls":     "test-urls",
"TestURLS":     "test-urls",
"aURLTest":     "a-url-test",
"S":            "s",
"s":            "s",
"aS":           "a-s",
"Sa":           "sa",
"as":           "as",
"lowerUpper":   "lower-upper",
"DockerFoo": "docker-foo",
"FooBar": "foo-bar",
"FOOFooBAR": "foo-foo-bar",
"FoOoFooBARBaZ": "fo-oo-foo-bar-ba-z",
"FOOFOoBARBaZa": "foo-f-oo-bar-ba-za"
*/

func splitCamel(s string, sep byte) string {
	var splStr []byte
	sLen := len(s)
	// easier to handle short strings here:
	if sLen < 2 {
		return strings.ToLower(s)
	} else if sLen == 2 {
		if isLower(s[0]) && isUpper(s[1]) {
			return fmt.Sprintf("%c-%c", s[0], toLower(s[1]))
		}
		return strings.ToLower(s)
	}
	var appendLastCharacter []byte
	// the final letter doesn't follow the same rules (to support plurals like URLs)
	// TestURLs --> test-urls (Ending Multiword Caps + one lowercase)
	// Also catches TestT --> test-t
	if (isLower(s[sLen-1]) && isUpper(s[sLen-2]) && isUpper(s[sLen-3])) || (isLower(s[sLen-2]) && isUpper(s[sLen-1])) {
		if isUpper(s[sLen-1]) {
			appendLastCharacter = append(appendLastCharacter, sep)
		}
		appendLastCharacter = append(appendLastCharacter, s[sLen-1])
		s = s[0 : sLen-1]
		sLen--
	}
	inWord := false
	wordStart := 0
	for i := 0; i < sLen-1; i++ {
		l, m := s[i], s[i+1]
		if !inWord && isLower(l) && isUpper(m) { // aU -> a-u...
			wordStart = i - 1
			if wordStart < 0 {
				wordStart = 0
			}
			// do not continue (keep going).
		} else if !inWord {
			inWord = true
			wordStart = i
			continue
		}
		// now in word
		if sameCase(l, m) {
			continue
		}
		// URLStuff -> url-stuff -- use l as boundry instead of m (case of multi caps)
		end := i + 1
		if i-1 >= 0 && isLower(m) && isUpper(l) && isUpper(s[i-1]) {
			end--
		}
		//FOOFoo -> foo-foo
		splStr = append(splStr, s[wordStart:end]...)
		splStr = append(splStr, sep)
		wordStart = end
		inWord = (end == i) // if we moved back above, then we're still in a word.
	}
	if inWord {
		splStr = append(splStr, s[wordStart:]...)
	}
	if len(appendLastCharacter) > 0 {
		splStr = append(splStr, appendLastCharacter...)
	}
	return strings.ToLower(string(splStr))
}

// this is included for historical reference. Might be more compact...
// but lacks uh... readability.
func splitCamelOld(s string, sep byte) string {
	r := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i == len(s)-2 && isLower(c) && isUpper(s[i+1]) {
			r = append(r, c, sep, toLower(s[i+1]))
			break
		} else if i == len(s)-2 && isUpper(c) && isLower(s[i+1]) {
			if i > 0 && isLower(s[i-1]) {
				r = append(r, sep, toLower(c), toUpper(s[i+1]))
			} else {
				r = append(r, toLower(c), toUpper(s[i+1]))
			}
			break
		}
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
