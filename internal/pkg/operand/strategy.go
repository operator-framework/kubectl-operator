package operand

import (
	"flag"
	"fmt"
)

// DeletionStrategy describes how to handle operands on-cluster when deleting the associated operator.
type DeletionStrategy struct {
	Kind DeletionStrategyKind
}

var _ flag.Value = &DeletionStrategy{}

type DeletionStrategyKind string

const (
	// Cancel is the default deletion strategy: it will cancel the deletion operation if operands are on-cluster.
	Cancel DeletionStrategyKind = "cancel"
	// Ignore will ignore the operands when deleting the operator, in effect orphaning them.
	Ignore DeletionStrategyKind = "ignore"
	// Delete will delete the operands associated with the operator before deleting the operator, allowing finalizers to run.
	Delete DeletionStrategyKind = "delete"
	// None represents an invalid empty strategy
	None DeletionStrategyKind = ""
)

func (d *DeletionStrategy) Set(str string) error {
	d.Kind = DeletionStrategyKind(str)
	return d.Valid()
}

func (d *DeletionStrategy) String() string {
	if d.Kind == None {
		d.Kind = Cancel
	}
	return string(d.Kind)
}

func (d DeletionStrategy) Valid() error {
	switch d.Kind {
	// set default strategy to cancel
	case None:
		d.Kind = Cancel
		fallthrough
	case Cancel, Ignore, Delete:
		return nil
	}
	return fmt.Errorf("unknown deletion strategy %q", d.Kind)
}

func (d DeletionStrategy) Type() string {
	return "DeletionStrategy"
}
