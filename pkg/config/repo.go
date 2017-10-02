package config

import "time"

// BaseRepo provides access to base configs.
type BaseRepo interface {
	Get(name string) (*BaseConfig, error)
}

// BaseConfig is the entire space of available parameters.
type BaseConfig struct {
	name string
}

// UserRepo provides access to user configs.
type UserRepo interface {
	lifecycle

	Get(baseName, id string) (UserConfig, error)
}

// UserConfig is a users rendered config.
type UserConfig struct {
	baseID      string
	id          string
	rendered    map[string]interface{}
	ruleIDs     []string
	userID      string
	createdAt   time.Time
	activatedAt time.Time
}

type lifecycle interface {
	Setup() error
	Teardown() error
}
