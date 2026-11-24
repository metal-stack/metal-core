package reconcile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

// execute deletes and writes the entries of the plan.
func execute(ctx context.Context, log *slog.Logger, config *db.ConfigDB, p Plan) error {
	var errs []error
	for _, key := range p.Deletes {
		log.Info("delete entry", "key", keyString(key))
		if _, err := config.DeleteEntry(ctx, key); err != nil {
			errs = append(errs, fmt.Errorf("could not delete %s: %w", keyString(key), err))
		}
	}
	for _, w := range p.Writes {
		log.Info("write entry", "key", keyString(w.Key), "set", w.Set, "delete", w.Delete)
		if err := config.SetFields(ctx, w.Key, w.Set, w.Delete); err != nil {
			errs = append(errs, fmt.Errorf("could not write %s: %w", keyString(w.Key), err))
		}
	}
	return errors.Join(errs...)
}
