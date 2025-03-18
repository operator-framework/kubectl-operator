package action

import "errors"

var (
	ErrNoResourcesFound = errors.New("no resources found")
	ErrNameAndSelector  = errors.New("name cannot be provided when a selector is specified")
	ErrNoChange         = errors.New("no changes detected - extension already in desired state")
)
