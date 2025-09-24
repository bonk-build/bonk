// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/go-viper/mapstructure/v2"
)

// ToProtoValue wraps any value into a [structpb.Value].
func ToProtoValue(value any) (*structpb.Value, error) {
	return toProtoValue(reflect.ValueOf(value))
}

//nolint:wrapcheck
func toProtoValue(value reflect.Value) (*structpb.Value, error) {
	var err error

	switch value.Type().Kind() {
	// Pointers & interfaces just get dereferenced and passed through
	case reflect.Interface, reflect.Pointer:
		return toProtoValue(value.Elem())

	case reflect.Array, reflect.Slice:
		list := &structpb.ListValue{
			Values: make([]*structpb.Value, value.Len()),
		}
		for idx, val := range value.Seq2() {
			list.Values[idx.Int()], err = toProtoValue(val)
			if err != nil {
				return nil, err
			}
		}

		return structpb.NewListValue(list), nil

	case reflect.Map:
		str := &structpb.Struct{
			Fields: make(map[string]*structpb.Value, value.Len()),
		}
		for key, val := range value.Seq2() {
			str.Fields[fmt.Sprint(key)], err = toProtoValue(val)
			if err != nil {
				return nil, err
			}
		}

		return structpb.NewStructValue(str), nil

	// Structs should get passed through mapstructure
	case reflect.Struct:
		var intermediate map[string]any
		err = mapstructure.Decode(value.Interface(), &intermediate)
		if err != nil {
			return nil, fmt.Errorf("failed to decode args into mapstructure: %w", err)
		}

		return ToProtoValue(intermediate)

	default:
		return structpb.NewValue(value.Interface())
	}
}
