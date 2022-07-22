package gcp

import (
	"context"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"
	// "google.golang.org/api/option"

	//compute "cloud.google.com/go/compute/apiv1"
	//computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier string = "GCP"

// Client represents a GCP Client
type Client struct {
	projectID      string
	region         string
	zone           string
	instanceType   string
	computeService *computev1.Service
	tags           map[string]string
	logger         ocmlog.Logger
	output         output.Output
}

// type computepb interface {
// }
type MachineTypeClient interface {
	List(projectID string, zone string) *computev1.MachineTypesListCall
}
type InstancesClient interface {
	Get(projectID string, zone string, instanceName string) *computev1.InstancesGetCall
	SetLabels(projectID string, zone string, instanceName string, reqbody *computev1.InstancesSetLabelsRequest) *computev1.InstancesSetLabelsCall
	GetSerialPortOutput(projectID string, zone string, instanceName string) *computev1.InstancesGetSerialPortOutputCall
	Stop(projectID string, zone string, instanceName string) *computev1.InstancesStopCall
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
