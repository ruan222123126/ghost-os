package session

import "errors"

type StoreOptions struct {
	HumanLogFullEnabled func() bool
}

func resolveStoreOptions(options []StoreOptions) (StoreOptions, error) {
	switch len(options) {
	case 0:
		return StoreOptions{}, nil
	case 1:
		return options[0], nil
	default:
		return StoreOptions{}, errors.New("at most one StoreOptions value is supported")
	}
}
