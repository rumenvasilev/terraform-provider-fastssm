package tfresource

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type EmptyResultError struct {
	LastRequest interface{}
}

var ErrEmptyResult = &EmptyResultError{}

func NewEmptyResultError(lastRequest interface{}) error {
	return &EmptyResultError{
		LastRequest: lastRequest,
	}
}

func (e *EmptyResultError) Error() string {
	return "empty result"
}

func (e *EmptyResultError) Is(err error) bool {
	_, ok := err.(*EmptyResultError)
	return ok
}

func (e *EmptyResultError) As(target interface{}) bool {
	t, ok := target.(**retry.NotFoundError)
	if !ok {
		return false
	}

	*t = &retry.NotFoundError{
		Message:     e.Error(),
		LastRequest: e.LastRequest,
	}

	return true
}
