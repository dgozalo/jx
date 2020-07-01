package cloudformation

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/jenkins-x/jx/v2/pkg/log"
	"github.com/pkg/errors"
)

type cloudFormationHandler struct {
	cf cloudformationiface.CloudFormationAPI
}

// NewCloudFormationAPIHandler will return an cloudFormationHandler with configured credentials
func NewCloudFormationAPIHandler(awsSession client.ConfigProvider, cf ...cloudformationiface.CloudFormationAPI) (*cloudFormationHandler, error) {
	if len(cf) == 1 {
		return &cloudFormationHandler{
			cf: cf[0],
		}, nil
	}
	return &cloudFormationHandler{
		cf: cloudformation.New(awsSession),
	}, nil
}

func (o *cloudFormationHandler) DeleteStack(stackName *string) error {
	deletedStack, err := o.deleteStackAndWait(stackName)
	if err != nil {
		switch err.(type) {
		case awserr.Error:
			if err.(awserr.Error).Code() == "ResourceNotReady" {
				retainResources, err := o.getFailedDeletedResources(stackName)
				if err != nil {
					return err
				}
				deletedStack, err = o.deleteStackAndWait(stackName, retainResources...)
				if err != nil {
					return err
				}
				if *deletedStack.StackStatus != cloudformation.StackStatusDeleteFailed {
					return errors.New("unable to delete the stack after two attempts")
				}
			}
		default:
			return err
		}
	}
	return nil
}

func (o *cloudFormationHandler) deleteStackAndWait(stackName *string, retainedResources ...*string) (*cloudformation.Stack, error) {
	_, err := o.cf.DeleteStack(&cloudformation.DeleteStackInput{
		StackName:       stackName,
		RetainResources: retainedResources,
	})
	if err != nil {
		return nil, errors.Wrap(err, "there was a problem deleting the cloudformation stack")
	}

	err = o.cf.WaitUntilStackDeleteComplete(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})
	if err != nil {
		return nil, err
	}

	describedStack, err := o.DescribeStack(stackName)
	if err != nil {
		return nil, err
	}
	return describedStack, nil
}

func (o *cloudFormationHandler) ListStacks(filterFunc func(*cloudformation.Stack) bool) ([]*cloudformation.Stack, error) {
	var selectedStacks []*cloudformation.Stack
	describeStacksOutput, err := o.cf.DescribeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, errors.Wrap(err, "there was a problem listing cloudformation stacks")
	}
	for _, stack := range describeStacksOutput.Stacks {
		log.Logger().Debugf("Described stack %s", stack.StackName)
		if filterFunc == nil {
			selectedStacks = append(selectedStacks, stack)
		} else if filterFunc(stack) {
			selectedStacks = append(selectedStacks, stack)
		}
	}
	return selectedStacks, nil
}

func (o *cloudFormationHandler) DescribeStack(stackName *string) (*cloudformation.Stack, error) {
	describeOutput, err := o.cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})
	if err.(awserr.Error).Code() == cloudformation.ErrCodeStackInstanceNotFoundException {

	}
	if err != nil {
		return nil, errors.Wrap(err, "there was a problem describing the CloudFormation stack")
	}
	if len(describeOutput.Stacks) > 1 {
		return nil, errors.New("we couldn't find an unique stack to describe")
	}
	return describeOutput.Stacks[0], nil
}

func (o *cloudFormationHandler) getFailedDeletedResources(stackName *string) ([]*string, error) {
	describeStackResourcesOutput, err := o.cf.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: stackName,
	})
	if err != nil {
		return nil, err
	}
	var stackResources []*string
	for _, resource := range describeStackResourcesOutput.StackResources {
		if *resource.ResourceStatus == cloudformation.ResourceStatusDeleteFailed {
			stackResources = append(stackResources, resource.LogicalResourceId)
		}
	}
	return stackResources, nil
}

func handleAWSError(err error) (string, string) {
	switch err.(type) {
	case awserr.Error:
		awsError := err.(awserr.Error)
		return awsError.Code(), awsError.Message()
	default:
		return "", ""
	}
}
