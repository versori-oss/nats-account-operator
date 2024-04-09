package controllers

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	"github.com/versori-oss/nats-account-operator/pkg/apis"
)

// this test is written to demonstrate status equality
func Test_UserStatus_Compare(t *testing.T) {
	timeNow := time.Now()

	usr1 := v1alpha1.User{
		Status: v1alpha1.UserStatus{
			Status: v1alpha1.Status{
				Conditions: apis.Conditions{
					{
						Type:     apis.ConditionReady,
						Status:   corev1.ConditionTrue,
						Severity: apis.ConditionSeverityError,
						LastTransitionTime: apis.VolatileTime{
							Inner: metav1.NewTime(timeNow),
						},
						Reason:  "test",
						Message: "test",
					},
				},
			},
			KeyPair: &v1alpha1.KeyPair{
				PublicKey:      "public",
				SeedSecretName: "seed",
			},
			AccountRef: &v1alpha1.InferredObjectReference{
				Namespace: "ns",
				Name:      "name",
			},
		},
	}

	usr2 := v1alpha1.User{
		Status: v1alpha1.UserStatus{
			Status: v1alpha1.Status{
				Conditions: apis.Conditions{
					{
						Type:     apis.ConditionReady,
						Status:   corev1.ConditionTrue,
						Severity: apis.ConditionSeverityError,
						LastTransitionTime: apis.VolatileTime{
							// set different time to check if equality works
							Inner: metav1.NewTime(timeNow.Add(time.Second)),
						},
						Reason:  "test",
						Message: "test",
					},
				},
			},
			KeyPair: &v1alpha1.KeyPair{
				PublicKey:      "public",
				SeedSecretName: "seed",
			},
			AccountRef: &v1alpha1.InferredObjectReference{
				Namespace: "ns",
				Name:      "name",
			},
		},
	}

	if equality.Semantic.DeepEqual(&usr1.Status, usr2.Status) {
		t.Error("expected usr1.Status != usr2.Status")
	}

	if !equality.Semantic.DeepEqual(usr1.Status, usr2.Status) {
		t.Error("expected usr1.Status == usr2.Status")
	}
}
