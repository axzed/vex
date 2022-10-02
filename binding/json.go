// Copyright 2022 Xue WenChao. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

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
	IsValid               bool
}

func (b jsonBinding) Name() string {
	return "json"
}

func (b jsonBinding) Bind(r *http.Request, obj any) error {
	// POST param in the body
	body := r.Body
	if body == nil {
		return errors.New("invalid request!!!")
	}
	decoder := json.NewDecoder(body)
	// if you have unknown fields in request param json, this will handle it
	if b.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if b.IsValid {
		err := validateParam(obj, decoder)
		if err != nil {
			return err
		}
	} else {
		err := decoder.Decode(obj)
		if err != nil {
			return err
		}
	}
	return validate(obj)
}

// validateParam check the json param's validation
func validateParam(obj any, decoder *json.Decoder) error {
	// parse to map, then compare by key of map and struct
	// judge type by reflect
	valueOf := reflect.ValueOf(obj)
	// IsPointer ?
	if valueOf.Kind() != reflect.Pointer {
		return errors.New("This argumet must have a pointer type")
	}
	elem := valueOf.Elem().Interface()
	of := reflect.ValueOf(elem)
	// judge the type of value
	switch of.Kind() {
	case reflect.Struct:
		return checkParam(of, obj, decoder)
	case reflect.Slice, reflect.Array:
		elem := of.Type().Elem()
		if elem.Kind() == reflect.Struct {
			return checkParamSlice(elem, obj, decoder)
		}
	default:
		_ = decoder.Decode(obj)
	}
	return nil
}

// checkParamSlice is a method to handle the param in json and store in the slice or array
func checkParamSlice(of reflect.Type, obj any, decoder *json.Decoder) error {
	mapValue := make([]map[string]interface{}, 0)
	_ = decoder.Decode(&mapValue)
	for i := 0; i < of.NumField(); i++ {
		field := of.Field(i)
		name := field.Name
		// get the json name by json tag and key
		jsonName := field.Tag.Get("json")
		if jsonName != "" {
			name = jsonName
		}
		// self define the tag about vex framework
		required := field.Tag.Get("vex")
		for _, v := range mapValue {
			value := v[name]
			if value == nil && required == "required" {
				return errors.New(fmt.Sprintf("filed [%s] is not exist, because [%s] is required", jsonName, jsonName))
			}
		}
	}
	b, _ := json.Marshal(mapValue)
	_ = json.Unmarshal(b, obj)
	return nil
}

// checkParam is a method handle the param by tag and check the validation of json param in request body
func checkParam(of reflect.Value, obj any, decoder *json.Decoder) error {
	mapValue := make(map[string]interface{})
	_ = decoder.Decode(&mapValue)
	for i := 0; i < of.NumField(); i++ {
		field := of.Type().Field(i)
		name := field.Name
		// get the json name by json tag and key
		jsonName := field.Tag.Get("json")
		if jsonName != "" {
			name = jsonName
		}
		// self define the tag about vex framework
		required := field.Tag.Get("vex")
		value := mapValue[name]
		if value == nil && required == "required" {
			return errors.New(fmt.Sprintf("filed [%s] is not exist, because [%s] is required", jsonName, jsonName))
		}
	}
	b, _ := json.Marshal(mapValue)
	_ = json.Unmarshal(b, obj)
	return nil
}
