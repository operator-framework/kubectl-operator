package operand

import (
	"errors"
	"flag"
	"fmt"
)

// DeletionStrategy describes how to handle operands on-cluster when deleting the associated operator.
type DeletionStrategy string

var _ flag.Value = new(DeletionStrategy)

const (
	// Abort is the default deletion strategy: it will abort the deletion operation if operands are on-cluster.
	Abort DeletionStrategy = "abort"
	// Ignore will ignore the operands when deleting the operator, in effect orphaning them.
	Ignore DeletionStrategy = "ignore"
	// Delete will delete the operands associated with the operator before deleting the operator, allowing finalizers to run.
	Delete DeletionStrategy = "delete"
)

func (d *DeletionStrategy) Set(str string) error {
	*d = DeletionStrategy(str)
	return d.Valid()
}

func (d DeletionStrategy) String() string {
	return string(d)
}

func (d DeletionStrategy) Valid() error {
	switch d {
	case Abort, Ignore, Delete:
		return nil
	}
	return fmt.Errorf("unknown operand deletion strategy %q", d)
}

func (d DeletionStrategy) Type() string {
	return "DeletionStrategy"
}

var ErrAbortStrategy = errors.New(`operand deletion aborted: one or more operands exist and operand strategy is "abort"`)
