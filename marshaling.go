package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/dop251/goja"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func tfToGoja(r *goja.Runtime, v tftypes.Value) goja.Value {
	t := v.Type()
	switch {
	case v.IsNull():
		return goja.Null()
	case t.Is(tftypes.Bool):
		var b bool
		v.As(&b)
		return r.ToValue(b)
	case t.Is(tftypes.String):
		var s string
		v.As(&s)
		return r.ToValue(s)
	case t.Is(tftypes.Number):
		var bf big.Float
		v.As(&bf)
		f, _ := bf.Float64()
		return r.ToValue(f)
	}
	switch t.(type) {
	case tftypes.List:
	case tftypes.Set:
	case tftypes.Tuple:
		var tl []tftypes.Value
		v.As(&tl)
		gl := make([]any, len(tl))
		for i, tv := range tl {
			gl[i] = tfToGoja(r, tv)
		}
		return r.NewArray(gl...)
	case tftypes.Map:
	case tftypes.Object:
		var tm map[string]tftypes.Value
		v.As(&tm)
		gm := r.NewObject()
		for k, tv := range tm {
			gm.DefineDataProperty(k, tfToGoja(r, tv), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE)
		}
		return gm
	}
	return goja.Undefined()
}

func tfdynamicToGoja(r *goja.Runtime, dv *tfprotov6.DynamicValue) (goja.Value, error) {
	if tv, err := dv.Unmarshal(tftypes.DynamicPseudoType); err != nil {
		return nil, fmt.Errorf("Error unmarshaling value: %w", err)
	} else {
		return tfToGoja(r, tv), nil
	}
}

func examineListTvtypes(tvs []tftypes.Value) tftypes.Type {
	state := 0
	types := make([]tftypes.Type, len(tvs))
	var tvt tftypes.Type
	for i, tv := range tvs {
		t := tv.Type()
		types[i] = t
		if state == 0 {
			tvt = t
			state = 1
		} else if state == 1 && !tvt.Equal(t) {
			state = 2
		}
	}
	if state == 1 {
		return tftypes.List{ElementType: tvt}
	} else {
		return tftypes.Tuple{ElementTypes: types}
	}
}

func examineMapTvtypes(tvs map[string]tftypes.Value) tftypes.Type {
	state := 0
	types := make(map[string]tftypes.Type, len(tvs))
	var tvt tftypes.Type
	for k, tv := range tvs {
		t := tv.Type()
		types[k] = t
		if state == 0 {
			tvt = t
			state = 1
		} else if state == 1 && !tvt.Equal(t) {
			state = 2
		}
	}
	if state == 1 {
		return tftypes.Map{ElementType: tvt}
	} else {
		return tftypes.Object{AttributeTypes: types}
	}
}

func jsonToTf(dec *json.Decoder, end json.Delim) (tftypes.Value, bool, error) {
	// not doing type unification, that's huge overkill for a scripting plugin
	if token, err := dec.Token(); err != nil || token == nil {
		return tftypesNull, false, err
	} else if b, ok := token.(bool); ok {
		return tftypes.NewValue(tftypes.Bool, b), false, nil
	} else if s, ok := token.(string); ok {
		return tftypes.NewValue(tftypes.String, s), false, nil
	} else if f, ok := token.(float64); ok {
		return tftypes.NewValue(tftypes.Number, f), false, nil
	} else if delim, ok := token.(json.Delim); !ok {
		// unexpected token
	} else if delim == end {
		return tftypesNull, true, nil
	} else if delim == '[' {
		var tvs []tftypes.Value
		for {
			if tv, endarr, err := jsonToTf(dec, ']'); err != nil {
				return tftypesNull, false, err
			} else if endarr {
				return tftypes.NewValue(examineListTvtypes(tvs), tvs), false, nil
			} else {
				tvs = append(tvs, tv)
			}
		}
	} else if delim == '{' {
		tvs := make(map[string]tftypes.Value)
		for {
			if token, err = dec.Token(); err != nil {
				return tftypesNull, false, err
			} else if delim, ok := token.(json.Delim); ok && delim == '}' {
				return tftypes.NewValue(examineMapTvtypes(tvs), tvs), false, nil
			} else if k, ok := token.(string); !ok {
				return tftypesNull, false, fmt.Errorf("Missing json key, got token %v", token)
			} else if tv, _, err := jsonToTf(dec, 0); err != nil {
				return tftypesNull, false, err
			} else {
				tvs[k] = tv
			}
		}
	}
	return tftypesNull, false, fmt.Errorf("Unexpected token")
}

func gojaToTf(r *goja.Runtime, v goja.Value) (tftypes.Value, error) {
	if goja.IsNull(v) {
		return tftypesNull, nil
	}
	if b, err := v.ToObject(r).MarshalJSON(); err != nil {
		return tftypesNull, fmt.Errorf("Error marshaling js value to json: %w", err)
	} else if tv, _, err := jsonToTf(json.NewDecoder(bytes.NewReader(b)), 0); err != nil {
		return tftypesNull, fmt.Errorf("Error unmarshaling json to tf value: %w", err)
	} else {
		return tv, nil
	}
}

func gojaToTfdynamic(r *goja.Runtime, v goja.Value) (*tfprotov6.DynamicValue, error) {
	if tv, err := gojaToTf(r, v); err != nil {
		return nil, err
	} else if dv, err := tfprotov6.NewDynamicValue(tftypes.DynamicPseudoType, tv); err != nil {
		return nil, fmt.Errorf("Error wrapping tf value: %w", err)
	} else {
		return &dv, nil
	}
}
