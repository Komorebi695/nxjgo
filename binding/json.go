package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

type jsonBinding struct {
	DisallowUnknownFields bool
	IsValidate            bool
}

func (b jsonBinding) Name() string {
	return "json"
}

func (b jsonBinding) Bind(r *http.Request, obj any) error {
	body := r.Body
	if r == nil || body == nil {
		return errors.New("invalid request")
	}
	decoder := json.NewDecoder(body)
	if b.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if b.IsValidate {
		if err := validateRequireParam(obj, decoder); err != nil {
			return err
		}
	} else {
		if err := decoder.Decode(obj); err != nil {
			return err
		}
	}
	return validate(obj)
}

func validate(obj any) error {
	return Validator.ValidateStruct(obj)
}

func validateRequireParam(data interface{}, decoder *json.Decoder) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("the argument must be a non-nil pointer")
	}
	t := rv.Elem().Interface()
	of := reflect.ValueOf(t)
	switch of.Kind() {
	case reflect.Struct:
		if err := checkParam(of, data, decoder); err != nil {
			return err
		}
	case reflect.Slice, reflect.Array:
		elem := of.Type().Elem()
		elemType := elem.Kind()
		if elemType == reflect.Struct {
			if err := checkParamSlice(elem, data, decoder); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkParamSlice(emel reflect.Type, obj interface{}, decoder *json.Decoder) error {
	mapData := make([]map[string]interface{}, 0)
	if err := decoder.Decode(&mapData); err != nil {
		return err
	}
	if len(mapData) <= 0 {
		return nil
	}
	for i := 0; i < emel.NumField(); i++ {
		filed := emel.Field(i)
		tag := filed.Tag.Get("json")
		required := filed.Tag.Get("nxj")
		for _, v := range mapData {
			if v[tag] == nil && required == "required" {
				return errors.New(fmt.Sprintf("filed [%s] is not exist,because [%s] is required.", tag, tag))
			}
		}
	}
	marshal, err := json.Marshal(mapData)
	err = json.Unmarshal(marshal, obj)
	return err
}

func checkParam(of reflect.Value, obj interface{}, decoder *json.Decoder) error {
	mapData := make(map[string]interface{})
	if err := decoder.Decode(&mapData); err != nil {
		return err
	}
	for i := 0; i < of.NumField(); i++ {
		filed := of.Type().Field(i)
		tag := filed.Tag.Get("json")
		required := filed.Tag.Get("nxj")
		if mapData[tag] == nil && required == "required" {
			return errors.New(fmt.Sprintf("filed [%s] is not exist,because [%s] is required.", tag, tag))
		}
	}
	marshal, err := json.Marshal(mapData)
	err = json.Unmarshal(marshal, obj)
	return err
}
