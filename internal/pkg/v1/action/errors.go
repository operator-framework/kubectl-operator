package action

import "errors"

var (
	errNoChange = errors.New("no changes detected - operator already in desired state")
)
