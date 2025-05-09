package bundle_test

import (
	"testing"

	"github.com/limoges/gatepeeker/internal/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteYAMLEmpty(t *testing.T) {

	b := bundle.New()

	expected := ``

	buf, err := bundle.WriteYAML(b)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(buf))
}

func TestWriteYAML(t *testing.T) {

	policies := `
---
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: k8srequiredlabels
  annotations:
    metadata.gatekeeper.sh/title: "Required Labels"
    metadata.gatekeeper.sh/version: 1.1.2
    description: >-
      Requires resources to contain specified labels, with values matching
      provided regular expressions.
spec:
  crd:
    spec:
      names:
        kind: K8sRequiredLabels
      validation:
        openAPIV3Schema:
          type: object
          properties:
            message:
              type: string
            labels:
              type: array
              description: >-
                A list of labels and values the object must specify.
              items:
                type: object
                properties:
                  key:
                    type: string
                    description: >-
                      The required label.
                  allowedRegex:
                    type: string
                    description: >-
                      If specified, a regular expression the annotation's value
                      must match. The value must contain at least one match for
                      the regular expression.
  targets:
    - target: admission.k8s.gatekeeper.sh
      code:
      - engine: K8sNativeValidation
        source:
          validations:
          - expression: '(has(variables.anyObject.metadata) && variables.params.labels.all(entry, has(variables.anyObject.metadata.labels) && entry.key in variables.anyObject.metadata.labels))'
            messageExpression: '"missing required label, requires all of: " + variables.params.labels.map(entry, entry.key).join(", ")'
          - expression: '(has(variables.anyObject.metadata) && variables.params.labels.all(entry, has(variables.anyObject.metadata.labels) && entry.key in variables.anyObject.metadata.labels && (!has(entry.allowedRegex) || string(variables.anyObject.metadata.labels[entry.key]).matches(string(entry.allowedRegex)))))'
            message: "regex mismatch"
      - engine: Rego
        source:
          rego: |
            package k8srequiredlabels

            get_message(parameters, _default) := _default {
              not parameters.message
            }

            get_message(parameters, _) := parameters.message

            violation[{"msg": msg, "details": {"missing_labels": missing}}] {
              provided := {label | input.review.object.metadata.labels[label]}
              required := {label | label := input.parameters.labels[_].key}
              missing := required - provided
              count(missing) > 0
              def_msg := sprintf("you must provide labels: %v", [missing])
              msg := get_message(input.parameters, def_msg)
            }

            violation[{"msg": msg}] {
              value := input.review.object.metadata.labels[key]
              expected := input.parameters.labels[_]
              expected.key == key
              # do not match if allowedRegex is not defined, or is an empty string
              expected.allowedRegex != ""
              not regex.match(expected.allowedRegex, value)
              def_msg := sprintf("Label <%v: %v> does not satisfy allowed regex: %v", [key, value, expected.allowedRegex])
              msg := get_message(input.parameters, def_msg)
            }
---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sRequiredLabels
metadata:
  name: all-must-have-owner
spec:
  match:
    kinds:
      - apiGroups: [""]
        kinds: ["Namespace"]
  parameters:
    message: "All namespaces must have an owner label that points to your company username"
    labels:
      - key: owner
        allowedRegex: "^[a-zA-Z]+.agilebank.demo$"
`

	b, err := bundle.ParsePolicies([]byte(policies))
	require.NoError(t, err)

	expected := policies

	buf, err := bundle.WriteYAML(b)
	assert.NoError(t, err)
	assert.YAMLEq(t, expected, string(buf))
}
