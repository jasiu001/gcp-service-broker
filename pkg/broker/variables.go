// Copyright 2018 the Service Broker Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package broker

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"errors"

	"github.com/GoogleCloudPlatform/gcp-service-broker/pkg/validation"
	"github.com/GoogleCloudPlatform/gcp-service-broker/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/xeipuuv/gojsonschema"
)

const (
	JsonTypeString  JsonType = "string"
	JsonTypeNumeric JsonType = "number"
	JsonTypeInteger JsonType = "integer"
	JsonTypeBoolean JsonType = "boolean"
)

type JsonType string

type BrokerVariable struct {
	// Is this variable required?
	Required bool `yaml:"required,omitempty"`
	// The name of the JSON field this variable serializes/deserializes to
	FieldName string `yaml:"field_name"`
	// The JSONSchema type of the field
	Type JsonType `yaml:"type"`
	// Human readable info about the field.
	Details string `yaml:"details"`
	// The default value of the field.
	Default interface{} `yaml:"default,omitempty"`
	// If there are a limited number of valid values for this field then
	// Enum will hold them in value:friendly name pairs
	Enum map[interface{}]string `yaml:"enum,omitempty"`
	// Constraints holds JSON Schema validations defined for this variable.
	// Keys are valid JSON Schema validation keywords, and values are their
	// associated values.
	// http://json-schema.org/latest/json-schema-validation.html
	Constraints map[string]interface{} `yaml:"constraints,omitempty"`
}

var _ validation.Validatable = (*ServiceDefinition)(nil)

// Validate implements validation.Validatable.
func (bv *BrokerVariable) Validate() (errs *validation.FieldError) {
	return errs.Also(
		validation.ErrIfBlank(bv.FieldName, "field_name"),
		validation.ErrIfNotJSONSchemaType(string(bv.Type), "type"),
		validation.ErrIfBlank(bv.Details, "details"),
	)
}

// ToSchema converts the BrokerVariable into the value part of a JSON Schema.
func (bv *BrokerVariable) ToSchema() map[string]interface{} {
	schema := map[string]interface{}{}

	// Setting the auto-generated title comes first so it can be overridden
	// manually by constraints in special cases.
	if bv.FieldName != "" {
		schema[validation.KeyTitle] = fieldNameToLabel(bv.FieldName)
	}

	for k, v := range bv.Constraints {
		schema[k] = v
	}

	if len(bv.Enum) > 0 {
		enumeration := []interface{}{}
		for k, _ := range bv.Enum {
			enumeration = append(enumeration, k)
		}

		// Sort enumerations lexocographically for documentation consistency.
		sort.Slice(enumeration, func(i int, j int) bool {
			return fmt.Sprintf("%v", enumeration[i]) < fmt.Sprintf("%v", enumeration[j])
		})

		schema[validation.KeyEnum] = enumeration
	}

	if bv.Details != "" {
		schema[validation.KeyDescription] = bv.Details
	}

	if bv.Type != "" {
		schema[validation.KeyType] = bv.Type
	}

	if bv.Default != nil {
		schema[validation.KeyDefault] = bv.Default
	}

	return schema
}

func fieldNameToLabel(fieldName string) string {
	acronyms := map[string]string{
		"id":   "ID",
		"uri":  "URI",
		"url":  "URL",
		"gb":   "GB",
		"jdbc": "JDBC",
	}

	components := strings.FieldsFunc(fieldName, func(c rune) bool {
		return unicode.IsSpace(c) || c == '-' || c == '_' || c == '.'
	})

	for i, c := range components {
		if replace, ok := acronyms[c]; ok {
			components[i] = replace
		} else {
			components[i] = strings.ToUpper(c[:1]) + c[1:]
		}
	}

	return strings.Join(components, " ")
}

// Apply defaults adds default values for missing broker variables.
func ApplyDefaults(parameters map[string]interface{}, variables []BrokerVariable) {

	for _, v := range variables {
		if _, ok := parameters[v.FieldName]; !ok && v.Default != nil {
			parameters[v.FieldName] = v.Default
		}
	}

}

func ValidateVariables(parameters map[string]interface{}, variables []BrokerVariable) error {
	schema := CreateJsonSchema(variables)
	return ValidateVariablesAgainstSchema(parameters, schema)
}

// ValidateVariables validates a list of BrokerVariables are adhering to their JSONSchema.
func ValidateVariablesAgainstSchema(parameters map[string]interface{}, schema map[string]interface{}) error {

	result, err := gojsonschema.Validate(gojsonschema.NewGoLoader(schema), gojsonschema.NewGoLoader(parameters))
	if err != nil {
		return err
	}

	resultErrors := result.Errors()
	if len(resultErrors) == 0 {
		return nil
	}

	allErrors := &multierror.Error{
		ErrorFormat: utils.SingleLineErrorFormatter,
	}

	for _, r := range resultErrors {
		multierror.Append(allErrors, errors.New(r.String()))
	}

	return allErrors
}

// CreateJsonSchema outputs a JSONSchema given a list of BrokerVariables
func CreateJsonSchema(schemaVariables []BrokerVariable) map[string]interface{} {
	required := utils.NewStringSet()
	properties := make(map[string]interface{})

	for _, variable := range schemaVariables {
		properties[variable.FieldName] = variable.ToSchema()
		if variable.Required {
			required.Add(variable.FieldName)
		}
	}

	schema := map[string]interface{}{
		"$schema":    "http://json-schema.org/draft-04/schema#",
		"type":       "object",
		"properties": properties,
	}

	if !required.IsEmpty() {
		schema["required"] = required.ToSlice()
	}

	return schema
}
