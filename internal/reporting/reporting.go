package reporting

import (
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Report struct {
	results      map[string]*Result
	failureCount int
}

func New() *Report {
	r := &Report{}
	r.results = make(map[string]*Result)
	return r
}

func (r *Report) FailureCount() int {
	return r.failureCount
}

func (r *Report) AddResult(result *Result) {
	key := r.buildKey(result.Object)
	_, exists := r.results[key]
	if exists {
		panic("already exists")
	}
	r.results[key] = result
	r.failureCount += result.FailureCount()
}

func (r *Report) WriteTo(w io.Writer) {
	for key, value := range r.results {
		fmt.Fprintf(w, "%s %s\n", value.isValid(), key)
		for _, warning := range value.Warnings {
			fmt.Fprintf(w, "  WARNING %s\n", warning)
		}
		for _, deny := range value.Denials {
			fmt.Fprintf(w, "  FAILED %s\n", deny)
		}
	}
}

func ResourceName(obj *unstructured.Unstructured) string {
	apiVersion := obj.GetAPIVersion()
	kind := obj.GetKind()
	namespace := obj.GetNamespace()
	name := obj.GetName()
	return strings.ReplaceAll(fmt.Sprintf("%s:%s:%s:%s", apiVersion, kind, namespace, name), "/", ":")
}

func (r *Report) buildKey(obj *unstructured.Unstructured) string {
	return ResourceName(obj)
}

type Result struct {
	Object   *unstructured.Unstructured
	Warnings []string
	Denials  []string
}

func (r *Result) isValid() string {
	if len(r.Denials) > 0 {
		return "FAILED"
	}
	return "PASS"
}

func (r *Result) FailureCount() int {
	return len(r.Denials)
}
