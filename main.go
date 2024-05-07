package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	// NB: since I do not yet examine Javascript function prototypes,
	// all functions exported by this provider have an identical signature
	functionSignature = &tfprotov6.Function{
		VariadicParameter: &tfprotov6.FunctionParameter{
			AllowNullValue: true,
			Name:           "args",
			Type:           tftypes.DynamicPseudoType,
		},
		Return: &tfprotov6.FunctionReturn{
			Type: tftypes.DynamicPseudoType,
		},
	}

	tftypesNull = tftypes.NewValue(tftypes.String, nil)
)

type FunctionProvider struct {
	providerSchema *tfprotov6.Schema
	prog           *goja.Program
	functions      map[string]*tfprotov6.Function
	names          map[string]string
}

func createFunctionProvider(providerSchema *tfprotov6.Schema) *FunctionProvider {
	return &FunctionProvider{
		functions:      make(map[string]*tfprotov6.Function),
		names:          make(map[string]string),
		providerSchema: providerSchema,
	}
}

type providerConfig struct {
	Script string `tf:"js,required"`
	Strict bool   `tf:"strict"`
}

func (f *FunctionProvider) configureCore(config *tfprotov6.DynamicValue) (err error, summary string) {
	cfg := providerConfig{
		Strict: true,
	}
	if err = unmarshalDynamicValueToConfigType(&cfg, config); err != nil {
		return err, "Invalid configure payload"
	}
	if f.prog, err = goja.Compile("script.js", cfg.Script, cfg.Strict); err != nil {
		return err, "Failed to compile script"
	}
	js := goja.New()
	if _, err = js.RunProgram(f.prog); err != nil {
		return err, "Failed to compile script"
	}

	for _, k := range js.GlobalObject().Keys() {
		if _, ok := goja.AssertFunction(js.GlobalObject().Get(k)); ok && k[0] >= 'A' && k[0] <= 'Z' {
			tfname := strings.ToLower(k)
			f.functions[tfname] = functionSignature
			f.names[tfname] = k
		}
	}
	return nil, ""
}

func (f *FunctionProvider) callFunctionCore(fn string, args []*tfprotov6.DynamicValue) (*tfprotov6.DynamicValue, error) {
	js := goja.New()
	js.RunProgram(f.prog)

	gargs := make([]goja.Value, len(args))
	for i, arg := range args {
		if gv, err := tfdynamicToGoja(js, arg); err != nil {
			return nil, fmt.Errorf("Error marshaling argument #%d: %w", i, err)
		} else {
			gargs[i] = gv
		}
	}

	if jsfn, ok := f.names[fn]; !ok {
		return nil, fmt.Errorf("Unknown function %s", fn)
	} else if callable, ok := goja.AssertFunction(js.GlobalObject().Get(jsfn)); !ok {
		return nil, fmt.Errorf("Unknown function %s", fn) // should never happen by construction
	} else if gv, err := callable(goja.Undefined(), gargs...); err != nil {
		return nil, fmt.Errorf("Error calling %s(): %w", fn, err)
	} else if goja.IsUndefined(gv) {
		return nil, fmt.Errorf("Result is undefined (probably there was an error)")
	} else if dv, err := gojaToTfdynamic(js, gv); err != nil {
		return nil, fmt.Errorf("Error unmarshaling result: %w", err)
	} else {
		return dv, nil
	}
}

// -----------------------------
// function provider boilerplate
// -----------------------------
func (f *FunctionProvider) GetMetadata(context.Context, *tfprotov6.GetMetadataRequest) (*tfprotov6.GetMetadataResponse, error) {
	return &tfprotov6.GetMetadataResponse{
		ServerCapabilities: &tfprotov6.ServerCapabilities{GetProviderSchemaOptional: true},
	}, nil
}
func (f *FunctionProvider) GetProviderSchema(context.Context, *tfprotov6.GetProviderSchemaRequest) (*tfprotov6.GetProviderSchemaResponse, error) {
	return &tfprotov6.GetProviderSchemaResponse{
		ServerCapabilities: &tfprotov6.ServerCapabilities{GetProviderSchemaOptional: true},
		Provider:           f.providerSchema,
	}, nil
}
func (f *FunctionProvider) ValidateProviderConfig(ctx context.Context, req *tfprotov6.ValidateProviderConfigRequest) (*tfprotov6.ValidateProviderConfigResponse, error) {
	// Passthrough
	return &tfprotov6.ValidateProviderConfigResponse{PreparedConfig: req.Config}, nil
}
func (f *FunctionProvider) ConfigureProvider(ctx context.Context, req *tfprotov6.ConfigureProviderRequest) (*tfprotov6.ConfigureProviderResponse, error) {
	if err, msg := f.configureCore(req.Config); err != nil {
		return &tfprotov6.ConfigureProviderResponse{
			Diagnostics: []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
				Severity: tfprotov6.DiagnosticSeverityError,
				Summary:  msg,
				Detail:   err.Error(),
			}},
		}, nil
	} else {
		return &tfprotov6.ConfigureProviderResponse{}, nil
	}
}
func (f *FunctionProvider) StopProvider(context.Context, *tfprotov6.StopProviderRequest) (*tfprotov6.StopProviderResponse, error) {
	return &tfprotov6.StopProviderResponse{}, nil
}
func (f *FunctionProvider) ValidateResourceConfig(context.Context, *tfprotov6.ValidateResourceConfigRequest) (*tfprotov6.ValidateResourceConfigResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) UpgradeResourceState(context.Context, *tfprotov6.UpgradeResourceStateRequest) (*tfprotov6.UpgradeResourceStateResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ReadResource(context.Context, *tfprotov6.ReadResourceRequest) (*tfprotov6.ReadResourceResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) PlanResourceChange(context.Context, *tfprotov6.PlanResourceChangeRequest) (*tfprotov6.PlanResourceChangeResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ApplyResourceChange(context.Context, *tfprotov6.ApplyResourceChangeRequest) (*tfprotov6.ApplyResourceChangeResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ImportResourceState(context.Context, *tfprotov6.ImportResourceStateRequest) (*tfprotov6.ImportResourceStateResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ValidateDataResourceConfig(context.Context, *tfprotov6.ValidateDataResourceConfigRequest) (*tfprotov6.ValidateDataResourceConfigResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ReadDataSource(context.Context, *tfprotov6.ReadDataSourceRequest) (*tfprotov6.ReadDataSourceResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) CallFunction(ctx context.Context, req *tfprotov6.CallFunctionRequest) (*tfprotov6.CallFunctionResponse, error) {
	if _, ok := f.functions[req.Name]; ok {
		if res, err := f.callFunctionCore(req.Name, req.Arguments); err != nil {
			return &tfprotov6.CallFunctionResponse{
				Error: &tfprotov6.FunctionError{Text: err.Error()},
			}, nil
		} else {
			return &tfprotov6.CallFunctionResponse{
				Result: res,
			}, nil
		}
	}
	return nil, errors.New("unknown function " + req.Name)
}
func (f *FunctionProvider) GetFunctions(context.Context, *tfprotov6.GetFunctionsRequest) (*tfprotov6.GetFunctionsResponse, error) {
	return &tfprotov6.GetFunctionsResponse{
		Functions: f.functions,
	}, nil
}

func main() {
	if err := tf6server.Serve("registry.opentofu.org/opentofu/js", func() tfprotov6.ProviderServer {
		return createFunctionProvider(&tfprotov6.Schema{
			Block: &tfprotov6.SchemaBlock{
				Attributes: configTypeToSchemaAttributes[providerConfig](),
			},
		})
	}); err != nil {
		panic(err)
	}
}
