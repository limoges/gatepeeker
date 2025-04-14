package bundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-policy-agent/frameworks/constraint/pkg/core/templates"
	"github.com/open-policy-agent/gatekeeper/v3/apis"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/gator/reader"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var scheme = runtime.NewScheme()

func init() {
	if err := apis.AddToScheme(scheme); err != nil {
		panic(err)
	}
}

type Constraint struct {
	*unstructured.Unstructured
}

func newConstraint(obj *unstructured.Unstructured) *Constraint {
	return &Constraint{obj}
}

func (c *Constraint) resourceName() string {
	o := c.Unstructured
	apiVersion := o.GetAPIVersion()
	kind := o.GetKind()
	namespace := nilOrString(o.GetNamespace())
	name := nilOrString(o.GetName())
	return strings.ReplaceAll(fmt.Sprintf("%s:%s:%s:%s", apiVersion, kind, namespace, name), "/", ":")
}

func (c *Constraint) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Unstructured)
}

func (c *Constraint) GetObject() *unstructured.Unstructured {
	return c.Unstructured
}

type ConstraintTemplate struct {
	*templates.ConstraintTemplate
}

func newConstraintTemplate(t *templates.ConstraintTemplate) *ConstraintTemplate {
	return &ConstraintTemplate{t}
}

func (t *ConstraintTemplate) resourceName() string {
	o := t.ConstraintTemplate
	apiVersion := nilOrString(o.APIVersion)
	kind := nilOrString(o.Kind)
	namespace := nilOrString(o.GetNamespace())
	name := nilOrString(o.GetName())
	return strings.ReplaceAll(fmt.Sprintf("%s:%s:%s:%s", apiVersion, kind, namespace, name), "/", ":")
}

func (t *ConstraintTemplate) GetObject() *templates.ConstraintTemplate {
	return t.ConstraintTemplate
}

func (t *ConstraintTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.ConstraintTemplate)
}

type Bundle struct {
	constraints []*Constraint
	templates   []*ConstraintTemplate
}

func New() *Bundle {
	return new(Bundle)
}

func ParsePolicies(buf []byte) (*Bundle, error) {
	objects, err := reader.ReadK8sResources(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	var (
		constraints []*Constraint
		templates   []*ConstraintTemplate
	)

	for _, obj := range objects {
		switch {
		case reader.IsConstraint(obj):
			constraints = append(constraints, newConstraint(obj))
		case reader.IsTemplate(obj):
			t, err := reader.ToTemplate(scheme, obj)
			if err != nil {
				panic(err)
			}
			t.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind()) // reader.ToTemplate doesn't seem to set GroupVersionKind
			templates = append(templates, newConstraintTemplate(t))
		}
	}

	b := New()
	b.constraints = constraints
	b.templates = templates
	return b, nil
}

func (b *Bundle) Merge(other *Bundle) {
	b.constraints = append(b.constraints, other.constraints...)
	b.templates = append(b.templates, other.templates...)
}

func (b *Bundle) GetConstraints() []*Constraint {
	return b.constraints
}

func (b *Bundle) GetConstraintTemplates() []*ConstraintTemplate {
	return b.templates
}

func nilOrString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
func listTemplates(objects []*templates.ConstraintTemplate) (out []string) {
	for _, o := range objects {
		apiVersion := nilOrString(o.APIVersion)
		kind := nilOrString(o.Kind)
		namespace := nilOrString(o.GetNamespace())
		name := nilOrString(o.GetName())
		s := fmt.Sprintf("%s/%s:%s/%s", apiVersion, kind, namespace, name)
		out = append(out, s)
	}
	return out
}

var listConstraints = listResources

func listResources(objects []*unstructured.Unstructured) (out []string) {
	for _, o := range objects {
		apiVersion := o.GetAPIVersion()
		kind := o.GetKind()
		namespace := nilOrString(o.GetNamespace())
		name := nilOrString(o.GetName())
		s := fmt.Sprintf("%s/%s:%s/%s", apiVersion, kind, namespace, name)
		out = append(out, s)
	}
	return out
}
