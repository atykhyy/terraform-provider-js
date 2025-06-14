package main

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	goTypeKindToTfprotoType = map[reflect.Kind]tftypes.Type{
		reflect.Bool:    tftypes.Bool,
		reflect.Int:     tftypes.Number,
		reflect.Int8:    tftypes.Number,
		reflect.Int16:   tftypes.Number,
		reflect.Int32:   tftypes.Number,
		reflect.Int64:   tftypes.Number,
		reflect.Uint:    tftypes.Number,
		reflect.Uint8:   tftypes.Number,
		reflect.Uint16:  tftypes.Number,
		reflect.Uint32:  tftypes.Number,
		reflect.Uint64:  tftypes.Number,
		reflect.Float32: tftypes.Number,
		reflect.Float64: tftypes.Number,
		reflect.String:  tftypes.String,
	}
)

func configTypeToSchemaAttributes[T any]() (result []*tfprotov6.SchemaAttribute) {
	t := reflect.TypeFor[T]()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if tag, ok := f.Tag.Lookup("tf"); ok && f.IsExported() {
			parts := strings.Split(tag, ",")
			result = append(result, &tfprotov6.SchemaAttribute{
				Name:     parts[0],
				Type:     goTypeKindToTfprotoType[f.Type.Kind()],
				Required: slices.Contains(parts[1:], "required"),
				Optional: slices.Contains(parts[1:], "optional"),
			})
		}
	}
	return
}

func unmarshalDynamicValueToConfigType[T any](v *T, config *tfprotov6.DynamicValue) error {
	t := reflect.TypeFor[T]()
	o := tftypes.Object{}
	o.AttributeTypes = make(map[string]tftypes.Type)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if tag, ok := f.Tag.Lookup("tf"); ok && f.IsExported() {
			parts := strings.Split(tag, ",")
			o.AttributeTypes[parts[0]] = goTypeKindToTfprotoType[f.Type.Kind()]
		}
	}

	res, err := config.Unmarshal(o)
	if err != nil {
		return err
	}
	cfg := make(map[string]tftypes.Value)
	if err = res.As(&cfg); err != nil {
		return err
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if tag, ok := f.Tag.Lookup("tf"); ok && f.IsExported() {
			parts := strings.Split(tag, ",")
			name := parts[0]
			value, ok := cfg[name]
			if !ok && slices.Contains(parts[1:], "required") {
				return fmt.Errorf("Missing required configuration value '%s'", name)
			}
			if err = value.As(reflect.ValueOf(v).Elem().FieldByIndex(f.Index).Addr().Interface()); err != nil {
				return fmt.Errorf("Invalid value provided for '%s': %w", name, err)
			}
		}
	}
	return nil
}
