package s3

import (
	"fmt"

	common "jabberwocky238/combinator/core/common"
)

// S3Factory is a function that creates a S3 instance from a parsed URL
type S3Factory func(*ParsedS3URL) (common.S3, error)

var s3Factories = make(map[string]S3Factory)

// RegisterS3Factory registers a S3 factory for a specific type
func RegisterS3Factory(s3Type string, factory S3Factory) {
	s3Factories[s3Type] = factory
}

// CreateS3 creates a S3 instance based on the parsed URL
func CreateS3(parsed *ParsedS3URL) (common.S3, error) {
	factory, ok := s3Factories[parsed.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported S3 type: %s", parsed.Type)
	}
	return factory(parsed)
}
