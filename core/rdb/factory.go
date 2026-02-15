package rdb

import (
	"fmt"

	common "jabberwocky238/combinator/core/common"
)

// RDBFactory is a function that creates a RDB instance from a parsed URL
type RDBFactory func(*ParsedRDBURL, *common.NamespacedLogger) (common.RDB, error)

var rdbFactories = make(map[string]RDBFactory)

// RegisterRDBFactory registers a RDB factory for a specific type
func RegisterRDBFactory(rdbType string, factory RDBFactory) {
	rdbFactories[rdbType] = factory
}

// CreateRDB creates a RDB instance based on the parsed URL
func CreateRDB(parsed *ParsedRDBURL, log *common.NamespacedLogger) (common.RDB, error) {
	factory, ok := rdbFactories[parsed.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported RDB type: %s", parsed.Type)
	}
	return factory(parsed, log)
}
