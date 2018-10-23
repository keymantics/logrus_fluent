package logrus_fluent

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/tinylib/msgp/msgp"
)

type conversionConfiguration struct {
	tagName    string
	useMsgPack bool
}

// ConvertToValue make map data from struct and tags
func ConvertToValue(p interface{}, config conversionConfiguration) interface{} {

	if config.useMsgPack {
		switch p.(type) {
		case msgp.Marshaler:
			return p
		}
	}

	rv := toValue(p)
	switch rv.Kind() {
	case reflect.Struct:
		return convertFromStruct(rv.Interface(), config)
	case reflect.Map:
		return convertFromMap(rv, config)
	case reflect.Slice:
		return convertFromSlice(rv, config)
	case reflect.Chan:
		return nil
	case reflect.Invalid:
		return nil
	default:
		return rv.Interface()
	}
}

func convertFromMap(rv reflect.Value, config conversionConfiguration) interface{} {
	result := make(map[string]interface{})
	for _, key := range rv.MapKeys() {
		kv := rv.MapIndex(key)
		result[fmt.Sprint(key.Interface())] = ConvertToValue(kv.Interface(), config)
	}
	return result
}

func convertFromSlice(rv reflect.Value, config conversionConfiguration) interface{} {
	var result []interface{}
	for i, max := 0, rv.Len(); i < max; i++ {
		result = append(result, ConvertToValue(rv.Index(i).Interface(), config))
	}
	return result
}

// convertFromStruct converts struct to value
// see: https://github.com/fatih/structs/
func convertFromStruct(p interface{}, config conversionConfiguration) interface{} {
	result := make(map[string]interface{})
	return convertFromStructDeep(result, config, toType(p), toValue(p))
}

func convertFromStructDeep(result map[string]interface{}, config conversionConfiguration, t reflect.Type, values reflect.Value) interface{} {
	for i, max := 0, t.NumField(); i < max; i++ {
		f := t.Field(i)
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}

		if f.Anonymous {
			tt := f.Type
			if tt.Kind() == reflect.Ptr {
				tt = tt.Elem()
			}
			vv := values.Field(i)
			if !vv.IsValid() {
				continue
			}
			if vv.Kind() == reflect.Ptr {
				vv = vv.Elem()
			}

			if vv.Kind() == reflect.Struct {
				convertFromStructDeep(result, config, tt, vv)
			}
			continue
		}

		tag, opts := parseTag(f, config.tagName)
		if tag == "-" {
			continue // skip `-` tag
		}

		if !values.IsValid() {
			continue
		}
		v := values.Field(i)
		if opts.Has("omitempty") && isZero(v) {
			continue // skip zero-value when omitempty option exists in tag
		}
		name := getNameFromTag(f, config.tagName)
		result[name] = ConvertToValue(v.Interface(), config)
	}
	return result
}

// toValue converts any value to reflect.Value
func toValue(p interface{}) reflect.Value {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// toType converts any value to reflect.Type
func toType(p interface{}) reflect.Type {
	t := reflect.ValueOf(p).Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// isZero checks the value is zero-value or not
func isZero(v reflect.Value) bool {
	zero := reflect.Zero(v.Type()).Interface()
	value := v.Interface()
	return reflect.DeepEqual(value, zero)
}

// getNameFromTag return the value in tag or field name in the struct field
func getNameFromTag(f reflect.StructField, tagName string) string {
	tag, _ := parseTag(f, tagName)
	if tag != "" {
		return tag
	}
	return f.Name
}

// getTagValues returns tag value of the struct field
func getTagValues(f reflect.StructField, tag string) string {
	return f.Tag.Get(tag)
}

// parseTag returns the first tag value of the struct field
func parseTag(f reflect.StructField, tag string) (string, options) {
	return splitTags(getTagValues(f, tag))
}

// splitTags returns the first tag value and rest slice
func splitTags(tags string) (string, options) {
	res := strings.Split(tags, ",")
	return res[0], res[1:]
}

// TagOptions is wrapper struct for rest tag values
type options []string

// Has checks the value exists in the rest values or not
func (t options) Has(tag string) bool {
	for _, opt := range t {
		if opt == tag {
			return true
		}
	}
	return false
}
