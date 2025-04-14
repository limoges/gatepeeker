package validating

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/limoges/gatepeeker/internal/bundle"
	"github.com/limoges/gatepeeker/internal/reporting"
	opaclient "github.com/open-policy-agent/frameworks/constraint/pkg/client"
	"github.com/open-policy-agent/frameworks/constraint/pkg/client/drivers/rego"
	"github.com/open-policy-agent/frameworks/constraint/pkg/client/reviews"
	rtypes "github.com/open-policy-agent/frameworks/constraint/pkg/types"
	"github.com/open-policy-agent/gatekeeper/v3/apis"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/drivers/k8scel"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/gator"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/gator/reader"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/target"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/util"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var k8starget = &target.K8sValidationTarget{}

var scheme = runtime.NewScheme()

func init() {
	err := apis.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

}

type Client struct {
	client gator.Client
	bundle *bundle.Bundle
}

func NewClientWithBundle(ctx context.Context, b *bundle.Bundle) (*Client, error) {
	client, err := newGatorClient()
	if err != nil {
		return nil, err
	}

	for _, v := range b.GetConstraintTemplates() {
		responses, err := client.AddTemplate(ctx, v.GetObject())
		if err != nil {
			panic(fmt.Sprintf("failed to add template: %s", err))
		}

		for _, result := range responses.Results() {
			fmt.Println(result.Target, result.Msg)
		}
	}

	for _, v := range b.GetConstraints() {
		responses, err := client.AddConstraint(ctx, v.GetObject())
		if err != nil {
			panic(fmt.Sprintf("failed to add constraint: %s", err))
		}

		for _, result := range responses.Results() {
			fmt.Println(result.Target, result.Msg)
		}
	}

	c := &Client{}
	c.client = client
	c.bundle = b
	return c, nil
}

func newGatorClient() (gator.Client, error) {
	regoDriver, err := rego.New(rego.Tracing(true), rego.GatherStats(), rego.PrintEnabled(true), rego.Defaults())
	if err != nil {
		return nil, err
	}
	k8sDriver, err := k8scel.New()
	if err != nil {
		return nil, err
	}
	var opts []opaclient.Opt
	opts = append(
		opts,
		opaclient.Targets(k8starget),
		opaclient.Driver(regoDriver),
		opaclient.Driver(k8sDriver),
		opaclient.EnforcementPoints(util.WebhookEnforcementPoint),
	)
	return opaclient.NewClient(opts...)
}

func (c *Client) Validate(ctx context.Context, manifestsYAML []byte) (*reporting.Report, error) {

	if c.bundle == nil {
		return nil, errors.New("no constraints or templates to validate")
	}

	if len(c.bundle.GetConstraints()) == 0 {
		return nil, errors.New("no constraints to validate")
	}
	if len(c.bundle.GetConstraintTemplates()) == 0 {
		return nil, errors.New("no templates to validate")
	}

	resources, err := reader.ReadK8sResources(bytes.NewReader(manifestsYAML))
	if err != nil {
		panic(err)
	}

	report := reporting.New()
	for _, v := range resources {
		req, err := unstructuredToAdmissionRequest(v)
		if err != nil {
			panic(err)
		}

		resp, err := c.client.Review(ctx, req, reviews.EnforcementPoint(util.WebhookEnforcementPoint), reviews.Tracing(true))
		if err != nil {
			panic(fmt.Sprintf("failed review: %s", err))
		}

		denials, warnings := getValidationMessages(resp.Results(), req)
		result := &reporting.Result{}
		result.Object = v
		result.Denials = denials
		result.Warnings = warnings
		report.AddResult(result)
	}

	return report, nil
}

func unstructuredToAdmissionRequest(obj *unstructured.Unstructured) (*admissionv1.AdmissionRequest, error) {
	resourceJSON, err := obj.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("%w: unable to marshal JSON encoding of object", err)
	}

	req := &admissionv1.AdmissionRequest{
		Kind: metav1.GroupVersionKind{
			Group:   obj.GetObjectKind().GroupVersionKind().Group,
			Version: obj.GetObjectKind().GroupVersionKind().Version,
			Kind:    obj.GetObjectKind().GroupVersionKind().Kind,
		},
		Object: runtime.RawExtension{
			Raw: resourceJSON,
		},
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	return req, nil
}

func getValidationMessages(results []*rtypes.Result, req *admissionv1.AdmissionRequest) (deny, warn []string) {
	for _, result := range results {
		constraint := fmt.Sprintf("%s/%s/%s:%s",
			result.Constraint.GroupVersionKind().Group,
			result.Constraint.GroupVersionKind().Version,
			result.Constraint.GetKind(),
			result.Constraint.GetName(),
		)
		resource := fmt.Sprintf("%s/%s/%s:%s/%s",
			req.Kind.Group,
			req.Kind.Version,
			req.Kind.Kind,
			req.Namespace,
			req.Name,
		)

		action := result.EnforcementAction
		switch action {
		case "deny":
			msg := fmt.Sprintf("%s %s %s: %s (%s)\n", constraint, action, resource, result.Msg, result.Target)
			deny = append(deny, msg)
		case "warn":
			msg := fmt.Sprintf("%s %s %s: %s (%s)\n", constraint, action, resource, result.Msg, result.Target)
			warn = append(warn, msg)
		default:
			panic(fmt.Sprintf("unknown r.EnforcementAction: %s", action))
		}
	}
	return
}
