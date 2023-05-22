/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package apis provides common types and functions for interacting with CRD types. This has been copied from
// knative.dev/pkg and copyright/licence information has been preserved.
//
// Changes to upstream:
// - Disable deep-copy generation for interface types.
//
// +k8s:deepcopy-gen=package
package apis
