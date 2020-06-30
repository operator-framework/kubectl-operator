package operatorgroup

import (
	v1 "github.com/operator-framework/api/pkg/operators/v1"
)

type OperatorGroup v1.OperatorGroup

type Option func(*v1.OperatorGroup) error

func NewOperatorGroup(og *v1.OperatorGroup, opts ...Option) (*OperatorGroup, error) {

}
