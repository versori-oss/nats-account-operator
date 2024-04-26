package controllers

import (
	"fmt"

	"github.com/go-faster/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

type markConditionFunc func(reason, messageFormat string, messageA ...interface{})

// AsResult uses err to return a ctrl.Result and error to the reconciler runtime, taking
// into account whether the error is temporary, terminal or neither. The fallback is to return an
// empty result with the causing error, resulting in the reconciler runtime logging the error and
// re-enqueuing the reconciliation.
func AsResult(err error) (ctrl.Result, error) {
	if err == nil {
		return ctrl.Result{}, nil
	}

	if rerr, ok := errors.Into[*resultError](err); ok {
		return rerr.Result(), nil
	}

	return ctrl.Result{}, err
}

// MarkCondition uses err to mark the condition of the owning object. If err contains a conditionError,
// the condition will be marked as either failed or unknown, depending on the state of said conditionError.
func MarkCondition(err error, failure, unknown markConditionFunc) {
	if cerr, ok := errors.Into[*conditionError](err); ok {
		cerr.MarkCondition(failure, unknown)
	} else {
		unknown(v1alpha1.ReasonUnknownError, err.Error())
	}
}

type conditionError struct {
	failure bool
	reason  string
	err     error
}

func ConditionFailed(reason, msgFmt string, args ...any) error {
	return &conditionError{
		failure: true,
		reason:  reason,
		err:     fmt.Errorf(msgFmt, args...),
	}
}

func ConditionUnknown(reason, msgFmt string, args ...any) error {
	return &conditionError{
		failure: false,
		reason:  reason,
		err:     fmt.Errorf(msgFmt, args...),
	}
}

func (c *conditionError) Error() string {
	// use Errorf to allow msgFmt to contain %w for wrapping other errors.
	return c.err.Error()
}

func (c *conditionError) MarkCondition(failure, unknown markConditionFunc) {
	if c.failure {
		failure(c.reason, c.Error())
	} else {
		unknown(c.reason, c.Error())
	}
}

// Unwrap returns the underlying error.
func (c *conditionError) Unwrap() error {
	// use Errorf to allow msgFmt to contain %w for wrapping other errors.
	return c.err
}

// resultError is used to terminate reconciliation and control whether the request should be re-enqueued. This error
// should only be used after any causing errors have been handled/logged, they will not be reported to the reconciler
// runtime.
type resultError struct {
	result ctrl.Result
	cause  error
}

// NewResultError is used to terminate reconciliation and control whether the request should be re-enqueued by the
// result parameter.
func NewResultError(result ctrl.Result, err error) error {
	if err == nil {
		return nil
	}

	return &resultError{
		cause:  err,
		result: result,
	}
}

// TerminalError is used to terminate reconciliation and will not result in the request being re-enqueued.
func TerminalError(err error) error {
	if err == nil {
		return nil
	}

	return &resultError{
		cause: err,
	}
}

// TemporaryError is used to terminate reconciliation and will result in the request being re-enqueued.
func TemporaryError(err error) error {
	if err == nil {
		return nil
	}

	return &resultError{
		cause:  err,
		result: ctrl.Result{Requeue: true},
	}
}

func (r *resultError) Error() string {
	return r.cause.Error()
}

// Unwrap returns the underlying error.
func (r *resultError) Unwrap() error {
	return r.cause
}

func (r *resultError) Result() ctrl.Result {
	return r.result
}
