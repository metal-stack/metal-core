package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
)

// DefaultWaitTimeout bounds a single WaitFor, so one stuck port cannot use up the time of a whole reconfiguration.
const DefaultWaitTimeout = 20 * time.Second

type (
	// Condition is a state of the switch metal-core waits for, e.g. that orchagent removed a router interface.
	Condition struct {
		desc  string
		check CheckFunc
	}

	// CheckFunc reports whether a condition is met. The detail describes what is still pending and is only
	// shown while the condition is not met.
	CheckFunc func(ctx context.Context) (met bool, detail string, err error)
)

var errNotMet = errors.New("conditions not met")

func NewCondition(desc string, check CheckFunc) Condition {
	return Condition{desc: desc, check: check}
}

func (c Condition) String() string {
	return c.desc
}

// Pending returns the descriptions of all conditions that are not met.
func Pending(ctx context.Context, conds ...Condition) ([]string, error) {
	var pending []string
	for _, c := range conds {
		met, detail, err := c.check(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not check %s: %w", c, err)
		}
		if met {
			continue
		}
		if detail != "" {
			pending = append(pending, fmt.Sprintf("%s (%s)", c, detail))
		} else {
			pending = append(pending, c.String())
		}
	}
	return pending, nil
}

// WaitFor waits until all conditions are met and returns immediately if they already are.
// On timeout, the error lists the conditions that are still not met.
func WaitFor(ctx context.Context, log *slog.Logger, timeout time.Duration, conds ...Condition) error {
	if len(conds) == 0 {
		return nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var pending []string
	err := retry.Do(
		func() error {
			p, err := Pending(waitCtx, conds...)
			if err != nil {
				return err
			}
			pending = p
			if len(pending) > 0 {
				log.Debug("waiting for conditions", "pending", pending)
				return errNotMet
			}
			return nil
		},
		retry.Context(waitCtx),
		// errors while checking are not retried, they would be hidden by the timeout
		retry.RetryIf(func(err error) bool { return errors.Is(err, errNotMet) }),
		retry.Attempts(0),
		retry.Delay(250*time.Millisecond),
		retry.MaxDelay(2*time.Second),
	)
	// a check that the timeout cut off is reported as the timeout as well
	if err != nil && len(pending) > 0 && waitCtx.Err() != nil && ctx.Err() == nil {
		return fmt.Errorf("timed out after %s waiting for: %s", timeout, strings.Join(pending, ", "))
	}
	return err
}
