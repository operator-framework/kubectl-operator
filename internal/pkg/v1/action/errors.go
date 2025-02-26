package action

import "errors"

var (
	errNoResourcesFound = errors.New("no resources found")
	errNameAndSelector  = errors.New("name cannot be provided when a selector is specified")
)
