package tools

import (
	"github.com/invopop/jsonschema"
	"github.com/sofianhadi1983/anthropic-sdk-go"
)

func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T

	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Type:       "object",
		Properties: schema.Properties,
		Required:   schema.Required,
	}
}
