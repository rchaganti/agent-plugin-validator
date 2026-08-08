package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/ravik/agent-plugin-validator/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// Issue represents a single schema validation error or warning.
type Issue struct {
	Path    string `json:"path"`    // JSON Pointer path, e.g., "/name" or "/non-schema"
	Message string `json:"message"` // Human readable description
}

// Result holds the validation status, schema metadata, errors, and warnings.
type Result struct {
	Valid    bool               `json:"valid"`              // true if no fatal errors exist
	Schema   *schema.SchemaInfo `json:"schema"`             // metadata about the schema used
	Errors   []Issue            `json:"errors"`             // fatal validation error issues
	Warnings []Issue            `json:"warnings,omitempty"` // non-fatal warnings (e.g., unknown top-level fields)
}

type regexp2Wrapper struct {
	pattern string
	re      *regexp2.Regexp
}

func (w *regexp2Wrapper) MatchString(s string) bool {
	matched, _ := w.re.MatchString(s)
	return matched
}

func (w *regexp2Wrapper) String() string {
	return w.pattern
}

func compileRegexp2(regex string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(regex, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return &regexp2Wrapper{pattern: regex, re: re}, nil
}

// Validate validates a manifest JSON against the given schema bytes and metadata.
// Unknown top-level fields are treated as non-fatal warnings per Agent Plugins Spec §5.2.
func Validate(manifestBytes []byte, schemaInfo *schema.SchemaInfo) (*Result, error) {
	res := &Result{
		Valid:    false,
		Schema:   schemaInfo,
		Errors:   []Issue{},
		Warnings: []Issue{},
	}

	// 1. Verify JSON syntax of manifest
	var rawJSON interface{}
	if err := json.Unmarshal(manifestBytes, &rawJSON); err != nil {
		res.Errors = append(res.Errors, Issue{
			Path:    "/",
			Message: fmt.Sprintf("invalid JSON syntax: %v", err),
		})
		return res, nil
	}

	// 2. Unmarshal schema JSON for compiler
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaInfo.RawSchema))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema JSON: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.UseRegexpEngine(compileRegexp2)

	schemaName := "schema.json"
	if err := compiler.AddResource(schemaName, schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to load schema into compiler: %w", err)
	}

	compiledSchema, err := compiler.Compile(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSON schema: %w", err)
	}

	// 3. Unmarshal manifest for jsonschema validation
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(manifestBytes))
	if err != nil {
		res.Errors = append(res.Errors, Issue{
			Path:    "/",
			Message: fmt.Sprintf("failed to parse JSON for validation: %v", err),
		})
		return res, nil
	}

	// 4. Perform validation
	if err := compiledSchema.Validate(v); err != nil {
		if valErr, ok := err.(*jsonschema.ValidationError); ok {
			res.Errors, res.Warnings = processErrors(valErr)
		} else {
			res.Errors = append(res.Errors, Issue{
				Path:    "/",
				Message: err.Error(),
			})
		}
	}

	// Valid if 0 fatal errors (warnings do not cause validation failure)
	res.Valid = len(res.Errors) == 0
	return res, nil
}

// processErrors flattens jsonschema.ValidationError hierarchy and classifies
// root-level unknown fields as warnings (per Spec §5.2) and others as errors.
func processErrors(ve *jsonschema.ValidationError) ([]Issue, []Issue) {
	var errors []Issue
	var warnings []Issue
	walkError(ve, &errors, &warnings)
	return errors, warnings
}

func walkError(ve *jsonschema.ValidationError, errors *[]Issue, warnings *[]Issue) {
	if len(ve.Causes) == 0 {
		path := "/"
		if len(ve.InstanceLocation) > 0 {
			path = "/" + strings.Join(ve.InstanceLocation, "/")
		}

		// Top-level unknown property check (root object level)
		if len(ve.InstanceLocation) == 0 {
			if addErr, ok := ve.ErrorKind.(*kind.AdditionalProperties); ok {
				for _, prop := range addErr.Properties {
					*warnings = append(*warnings, Issue{
						Path:    "/" + prop,
						Message: fmt.Sprintf("unknown top-level field '%s' (ignored per spec §5.2)", prop),
					})
				}
				return
			}
		}

		msg := ve.Error()
		if idx := strings.Index(msg, " at "); idx != -1 {
			msg = msg[:idx]
		}

		*errors = append(*errors, Issue{
			Path:    path,
			Message: msg,
		})
		return
	}

	for _, cause := range ve.Causes {
		walkError(cause, errors, warnings)
	}
}
