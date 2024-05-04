package main

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"
)

func protoToCty(arg *tfprotov6.DynamicValue) (cty.Value, error) {
	// Decode using cty directly as it supports DynamicPseudoType
	// This is inspired by github.com/apparentlymart/go-tf-func-provider
	if len(arg.MsgPack) != 0 {
		return ctymsgpack.Unmarshal(arg.MsgPack, cty.DynamicPseudoType)
	}
	if len(arg.JSON) != 0 {
		return ctyjson.Unmarshal(arg.JSON, cty.DynamicPseudoType)
	}
	panic("unknown encoding")
}

func protoToJson(arg *tfprotov6.DynamicValue) ([]byte, error) {
	if ctyVal, err := protoToCty(arg); err != nil {
		return nil, err
	} else {
		return json.Marshal(ctyjson.SimpleJSONValue{ctyVal})
	}
}

func ctyToProto(ctyVal cty.Value) (*tfprotov6.DynamicValue, error) {
	result, err := ctymsgpack.Marshal(ctyVal, cty.DynamicPseudoType)
	if err != nil {
		return nil, err
	}
	return &tfprotov6.DynamicValue{
		MsgPack: result,
	}, nil
}

func jsonToProto(arg string) (*tfprotov6.DynamicValue, error) {
	var ctyVal ctyjson.SimpleJSONValue
	if err := json.Unmarshal([]byte(arg), &ctyVal); err != nil {
		return nil, err
	}
	return ctyToProto(ctyVal.Value)
}
