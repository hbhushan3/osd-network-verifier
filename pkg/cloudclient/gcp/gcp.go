package gcp

import (
	"context"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"
	// "google.golang.org/api/option"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier string = "GCP"

// Client represents a GCP Client
type Client struct {
	projectID      string
	region         string
	zone           string
	instanceType   string
	computeService ComputeServiceClient
	tags           map[string]string
	logger         ocmlog.Logger
	output         output.Output
}

type ComputeServiceClient interface {
	MachineTypes() *computev1.MachineTypesService
	Instances() *computev1.InstancesService
	NewInstancesRESTClient() *compute.InstancesClient
	// NewInstancesRESTClient(ctx context.Context, params *compute.NewInstancesRESTClient) error
	// Insert(ctx context.Context, params *compute.NewInstancesRESTClient.Insert) (*ec2.RunInstancesOutput, error)
	// Get(ctx context.Context, input *ec2.DescribeInstanceStatusInput) (*ec2.DescribeInstanceStatusOutput, error)
	// List(ctx context.Context, input *ec2.DescribeInstanceTypesInput) (*ec2.DescribeInstanceTypesOutput, error)
	// GetSerialPortOutput(ctx context.Context, input *ec2.GetConsoleOutputInput) (*ec2.GetConsoleOutputOutput, error)
	// Stop(ctx context.Context, input *ec2.TerminateInstancesInput) (*ec2.TerminaateInstancesOutput, error)
	// DescribeVpcAttribute(ctx context.Context, input *ec2.DescribeVpcAttributeInput) (*ec2.DescribeVpcAttributeOutput, error)
}

// type computepb interface {
// }
type MachineTypeClient interface {
	List(projectID string, zone string) *computev1.MachineTypesListCall
}
type InstancesClient interface {
	Get(projectID string, zone string, instanceName string) *InstancesGetCall
	SetLabels(projectID string, zone string, instanceName string, reqbody *compute.InstancesSetLabelsRequest) *InstancesSetLabelsCall
	GetSerialPortOutput(projectID string, zone string, instanceName string) *
	Stop(projectID string, zone string, instanceName string)
}

func (c *Client) ByoVPCValidator(ctx context.Context) error {
	c.logger.Info(ctx, "interface executed: %s", ClientIdentifier)
	return nil
}

func (c *Client) ValidateEgress(ctx context.Context, vpcSubnetID, cloudImageID string, kmsKeyID string, timeout time.Duration) *output.Output {
	return c.validateEgress(ctx, vpcSubnetID, cloudImageID, kmsKeyID, timeout)
}

func (c *Client) VerifyDns(ctx context.Context, vpcID string) *output.Output {
	return &c.output
}

func NewClient(ctx context.Context, logger ocmlog.Logger, credentials *google.Credentials, region, instanceType string, tags map[string]string) (*Client, error) {
	// initialize actual client
	return newClient(ctx, logger, credentials, region, instanceType, tags)
}
