package gcp

import (
	"context"
	"testing"
	// "time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	// "golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"

	"github.com/openshift/osd-network-verifier/pkg/cloudclient/mocks"
	// "github.com/openshift/osd-network-verifier/pkg/errors"
	// "github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
)

func TestByoVPCValidator(t *testing.T) {
	ctx := context.TODO()
	logger := &ocmlog.StdLogger{}
	client := &Client{logger: logger}
	err := client.ByoVPCValidator(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateEgress(t *testing.T) {
	// testID := "example-inst"
	// vpcSubnetID, cloudImageID := "dummy-id", "dummy-id"
	// consoleOut := `[   48.062407] cloud-init[2472]: Cloud-init v. 19.3-44.amzn2 running 'modules:final' at Mon, 07 Feb 2022 12:30:22 +0000. Up 48.00 seconds.
	// [   48.077429] cloud-init[2472]: USERDATA BEGIN
	// [   48.138248] cloud-init[2472]: USERDATA END`

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	FakeEC2Cli := mocks.NewMockInstancesClient(ctrl)

	FakeEC2Cli.EXPECT().Insert(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&computev1.InstancesInsertCall{

	})

	// ctrll := gomock.NewController(t)
	// defer ctrll.Finish()
	// FakeMachineTypeCli := mocks.NewMockMachineTypeClient(ctrl)

	FakeEC2Cli.EXPECT().Get(gomock.Any(), gomock.Any(),gomock.Any()).Times(1).Return(&computev1.InstancesGetCall{

	})

	FakeEC2Cli.EXPECT().SetLabels(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&computev1.InstancesSetLabelsCall{

	})

	// encodedconsoleOut := base64.StdEncoding.EncodeToString([]byte(consoleOut))
	FakeEC2Cli.EXPECT().GetSerialPortOutput(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(&computev1.InstancesGetSerialPortOutputCall{
		// Output: consoleOut,
	})

	FakeEC2Cli.EXPECT().Stop(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(computev1.InstancesStopCall{

	})

	cli := Client{
		ec2Client: FakeEC2Cli,
		logger:    &logging.GlogLogger{},
	}

	if !cli.validateEgress(context.TODO(), vpcSubnetID, cloudImageID, "", time.Duration(1*time.Second)).IsSuccessful() {
		t.Errorf("validateEgress(): should pass")
	}
}

func TestNewClient(t *testing.T) {

	// credentials := &google.Credentials{ProjectID: "my-sample-project-191923"}
	// region := "superstable-region1-z"
	// instanceType := "test-instance"
	// tags := map[string]string{"osd-network-verifier": "owned"}

	ctrll := gomock.NewController(t)
	defer ctrll.Finish()
	FakeMachineTypeCli := mocks.NewMockMachineTypeClient(ctrll)

	FakeMachineTypeCli.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(&computev1.MachineTypesListCall{

	})

	// client, err := NewClient(ctx, logger, credentials, region, instanceType, tags)
	// if err != nil {
	// 	t.Errorf("unexpected error creating client: %v", err)
	// }
	// if client.projectID != credentials.ProjectID {
	// 	t.Errorf("unexpected project ID: %v", client.projectID)
	// }
	// if client.region != region {
	// 	t.Errorf("unexpected region: %v", client.region)
	// }
	// if client.tags["osd-network-verifier"] != "owned" {
	// 	t.Errorf("unexpected tags: %v", client.tags)
	// }
}
