package nxjstrings

import (
	"fmt"
	"reflect"
	"strings"
)

func JoinStrings(str ...any) string {
	var sb strings.Builder
	for _, v := range str {
		sb.WriteString(check(v))
	}
	return sb.String()
}

func check(v any) string {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return v.(string)
	default:
		return fmt.Sprintf("%v", v)
	}
}
