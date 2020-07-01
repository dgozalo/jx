package cloudformation

import "github.com/aws/aws-sdk-go/service/cloudformation"

// CloudFormationer is an interface that abstracts the CloudFormation API
//go:generate pegomock generate github.com/jenkins-x/jx/v2/pkg/cloud/amazon/cloudformation CloudFormationer -o mocks/cloudFormationerMock.go
type CloudFormationer interface {
	DeleteStack(stackName *string) error
	ListStacks(filterFunc func(*cloudformation.Stack) bool) ([]*cloudformation.Stack, error)
}
