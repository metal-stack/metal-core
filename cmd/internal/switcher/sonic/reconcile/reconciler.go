package reconcile

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/desired"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/object"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const (
	// settleTimeout is how long a reconciliation waits for SONiC once it cannot progress, so that the usual
	// change is done within one reconciliation. What takes longer is continued by the next one.
	settleTimeout = 2 * time.Second
	minPause      = 250 * time.Millisecond
	maxPause      = time.Second
)

type (
	// FRRApplier renders the FRR configuration and reloads FRR if it changed.
	FRRApplier interface {
		Apply(ctx context.Context, cfg *types.Conf) error
	}

	// PendingError lists what waits for SONiC. The next reconciliation continues with it.
	PendingError struct {
		Pending []Pending
	}

	Reconciler struct {
		log    *slog.Logger
		db     *db.DB
		reader *observe.Reader
		quirks quirks.SonicQuirks
		frr    FRRApplier

		// applied is the config FRR was applied with last, nil before the first time
		applied *types.Conf
		// pause waits between the steps of a reconciliation
		pause func(ctx context.Context, d time.Duration) error
		// settle is how long to wait for SONiC once nothing progresses
		settle time.Duration
	}
)

func (e *PendingError) Error() string {
	pending := make([]string, 0, len(e.Pending))
	for _, p := range e.Pending {
		pending = append(pending, p.String())
	}
	return "waiting for SONiC: " + strings.Join(pending, "; ")
}

// Option changes how a Reconciler works.
type Option func(*Reconciler)

// WithPause replaces how the reconciler waits between steps, e.g. to run fake SONiC daemons in tests.
func WithPause(pause func(ctx context.Context, d time.Duration) error) Option {
	return func(r *Reconciler) {
		r.pause = pause
	}
}

func New(log *slog.Logger, sonicDb *db.DB, kernel observe.Kernel, q quirks.SonicQuirks, frr FRRApplier, opts ...Option) *Reconciler {
	r := &Reconciler{
		log:    log,
		db:     sonicDb,
		reader: observe.NewReader(sonicDb, kernel),
		quirks: q,
		frr:    frr,
		pause:  sleep,
		settle: settleTimeout,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Apply reconciles the switch with the config, step by step, until SONiC applied everything or nothing
// progresses for a while. Then it returns a *PendingError with what is left.
func (r *Reconciler) Apply(ctx context.Context, cfg *types.Conf) error {
	r.configureLLDP(ctx)

	var (
		lastProgress = time.Now()
		pause        = minPause
	)
	for {
		p, frrChanged, err := r.step(ctx, cfg)
		if err != nil {
			return err
		}
		if p.Done() {
			return nil
		}

		if len(p.Writes) > 0 || len(p.Deletes) > 0 || frrChanged {
			lastProgress = time.Now()
			pause = minPause
		} else if time.Since(lastProgress) >= r.settle {
			return &PendingError{Pending: p.Pending}
		}

		if err := r.pause(ctx, pause); err != nil {
			return &PendingError{Pending: p.Pending}
		}
		pause = min(2*pause, maxPause)
	}
}

// step plans on a snapshot of the switch, executes the plan and applies FRR.
func (r *Reconciler) step(ctx context.Context, cfg *types.Conf) (Plan, bool, error) {
	s, err := r.reader.Read(ctx, object.Tables)
	if err != nil {
		return Plan{}, false, err
	}
	vtep, err := object.VTEP(s)
	if err != nil {
		return Plan{}, false, err
	}
	sw, err := desired.Build(cfg, r.quirks.VrfEntries, vtep)
	if err != nil {
		return Plan{}, false, err
	}
	current := object.Current(s, sw.Ports)
	if err := r.reader.ReadRoutes(ctx, s, RouteVrfs(sw.Objects, current, s)); err != nil {
		return Plan{}, false, err
	}

	p := NewPlan(sw.Objects, current, s, frrState{applied: r.applied})
	for _, pending := range p.Pending {
		r.log.Debug("pending", "key", keyString(pending.Key), "reason", pending.Reason)
	}
	if err := execute(ctx, r.log, r.db.Config, p); err != nil {
		return Plan{}, false, err
	}

	changed, err := r.applyFRR(ctx, Effective(cfg, sw, s, r.applied))
	if err != nil {
		return Plan{}, false, err
	}
	return p, changed, nil
}

// applyFRR applies FRR with the config and tells whether it changed.
func (r *Reconciler) applyFRR(ctx context.Context, eff *types.Conf) (bool, error) {
	changed := r.applied == nil || !cmp.Equal(r.applied, eff)
	if !changed {
		return false, nil
	}
	if err := r.frr.Apply(ctx, quirks.ApplyFRRTemplateQuirks(r.quirks, *eff)); err != nil {
		return false, fmt.Errorf("could not apply frr: %w", err)
	}
	r.applied = eff
	return true, nil
}

// configureLLDP is independent of the ports, so its failures are only logged.
func (r *Reconciler) configureLLDP(ctx context.Context) {
	ports, err := r.db.Config.GetPorts(ctx)
	if err != nil {
		r.log.Error("failed to configure lldp", "error", err)
		return
	}
	for _, err := range r.quirks.LLDPConfiguration.Configure(ctx, r.db.Config, ports) {
		r.log.Error("failed to configure lldp", "error", err)
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
