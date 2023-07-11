package controllers

import (
    "errors"
    "fmt"
)

type markConditionFunc func(reason, messageFormat string, messageA ...interface{})

type conditionErr struct {
    failure bool
    reason string
    msgFmt string
    args []any
}

func ConditionFailed(reason, msgFmt string, args ...any) error {
    return &conditionErr{
        failure: true,
        reason: reason,
        msgFmt: msgFmt,
        args: args,
    }
}

func ConditionUnknown(reason, msgFmt string, args ...any) error {
    return &conditionErr{
        failure: false,
        reason: reason,
        msgFmt: msgFmt,
        args: args,
    }
}

func (c *conditionErr) Error() string {
    return fmt.Sprintf(c.msgFmt, c.args...)
}

func (c *conditionErr) MarkCondition(failure, unknown markConditionFunc) {
    if c.failure {
        failure(c.reason, c.msgFmt, c.args...)
    } else {
        unknown(c.reason, c.msgFmt, c.args...)
    }
}

func asConditionError(err error) (*conditionErr, bool) {
    var ce *conditionErr
    if errors.As(err, &ce) {
        return ce, true
    }

    return nil, false
}
