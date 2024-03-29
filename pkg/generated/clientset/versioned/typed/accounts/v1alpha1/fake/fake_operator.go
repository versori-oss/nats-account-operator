/*
MIT License

Copyright (c) 2022 Versori Ltd

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeOperators implements OperatorInterface
type FakeOperators struct {
	Fake *FakeAccountsV1alpha1
	ns   string
}

var operatorsResource = schema.GroupVersionResource{Group: "accounts", Version: "v1alpha1", Resource: "operators"}

var operatorsKind = schema.GroupVersionKind{Group: "accounts", Version: "v1alpha1", Kind: "Operator"}

// Get takes name of the operator, and returns the corresponding operator object, and an error if there is any.
func (c *FakeOperators) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Operator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(operatorsResource, c.ns, name), &v1alpha1.Operator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Operator), err
}

// List takes label and field selectors, and returns the list of Operators that match those selectors.
func (c *FakeOperators) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.OperatorList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(operatorsResource, operatorsKind, c.ns, opts), &v1alpha1.OperatorList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.OperatorList{ListMeta: obj.(*v1alpha1.OperatorList).ListMeta}
	for _, item := range obj.(*v1alpha1.OperatorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested operators.
func (c *FakeOperators) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(operatorsResource, c.ns, opts))

}

// Create takes the representation of a operator and creates it.  Returns the server's representation of the operator, and an error, if there is any.
func (c *FakeOperators) Create(ctx context.Context, operator *v1alpha1.Operator, opts v1.CreateOptions) (result *v1alpha1.Operator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(operatorsResource, c.ns, operator), &v1alpha1.Operator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Operator), err
}

// Update takes the representation of a operator and updates it. Returns the server's representation of the operator, and an error, if there is any.
func (c *FakeOperators) Update(ctx context.Context, operator *v1alpha1.Operator, opts v1.UpdateOptions) (result *v1alpha1.Operator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(operatorsResource, c.ns, operator), &v1alpha1.Operator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Operator), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeOperators) UpdateStatus(ctx context.Context, operator *v1alpha1.Operator, opts v1.UpdateOptions) (*v1alpha1.Operator, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(operatorsResource, "status", c.ns, operator), &v1alpha1.Operator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Operator), err
}

// Delete takes name of the operator and deletes it. Returns an error if one occurs.
func (c *FakeOperators) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(operatorsResource, c.ns, name, opts), &v1alpha1.Operator{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeOperators) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(operatorsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.OperatorList{})
	return err
}

// Patch applies the patch and returns the patched operator.
func (c *FakeOperators) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Operator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(operatorsResource, c.ns, name, pt, data, subresources...), &v1alpha1.Operator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Operator), err
}
