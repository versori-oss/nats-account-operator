/*
MIT License

Copyright (c) 2024 Versori Ltd

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

// Code generated by client-gen-v0.29.3. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	scheme "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SigningKeysGetter has a method to return a SigningKeyInterface.
// A group's client should implement this interface.
type SigningKeysGetter interface {
	SigningKeys(namespace string) SigningKeyInterface
}

// SigningKeyInterface has methods to work with SigningKey resources.
type SigningKeyInterface interface {
	Create(ctx context.Context, signingKey *v1alpha1.SigningKey, opts v1.CreateOptions) (*v1alpha1.SigningKey, error)
	Update(ctx context.Context, signingKey *v1alpha1.SigningKey, opts v1.UpdateOptions) (*v1alpha1.SigningKey, error)
	UpdateStatus(ctx context.Context, signingKey *v1alpha1.SigningKey, opts v1.UpdateOptions) (*v1alpha1.SigningKey, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.SigningKey, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.SigningKeyList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.SigningKey, err error)
	SigningKeyExpansion
}

// signingKeys implements SigningKeyInterface
type signingKeys struct {
	client rest.Interface
	ns     string
}

// newSigningKeys returns a SigningKeys
func newSigningKeys(c *AccountsV1alpha1Client, namespace string) *signingKeys {
	return &signingKeys{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the signingKey, and returns the corresponding signingKey object, and an error if there is any.
func (c *signingKeys) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.SigningKey, err error) {
	result = &v1alpha1.SigningKey{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("signingkeys").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SigningKeys that match those selectors.
func (c *signingKeys) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.SigningKeyList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.SigningKeyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("signingkeys").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested signingKeys.
func (c *signingKeys) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("signingkeys").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a signingKey and creates it.  Returns the server's representation of the signingKey, and an error, if there is any.
func (c *signingKeys) Create(ctx context.Context, signingKey *v1alpha1.SigningKey, opts v1.CreateOptions) (result *v1alpha1.SigningKey, err error) {
	result = &v1alpha1.SigningKey{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("signingkeys").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(signingKey).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a signingKey and updates it. Returns the server's representation of the signingKey, and an error, if there is any.
func (c *signingKeys) Update(ctx context.Context, signingKey *v1alpha1.SigningKey, opts v1.UpdateOptions) (result *v1alpha1.SigningKey, err error) {
	result = &v1alpha1.SigningKey{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("signingkeys").
		Name(signingKey.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(signingKey).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *signingKeys) UpdateStatus(ctx context.Context, signingKey *v1alpha1.SigningKey, opts v1.UpdateOptions) (result *v1alpha1.SigningKey, err error) {
	result = &v1alpha1.SigningKey{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("signingkeys").
		Name(signingKey.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(signingKey).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the signingKey and deletes it. Returns an error if one occurs.
func (c *signingKeys) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("signingkeys").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *signingKeys) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("signingkeys").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched signingKey.
func (c *signingKeys) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.SigningKey, err error) {
	result = &v1alpha1.SigningKey{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("signingkeys").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
