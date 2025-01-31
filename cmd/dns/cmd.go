package dns

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
)

var (
	regionEnvVarStr string = "AWS_DEFAULT_REGION"
	regionDefault   string = "us-east-2"
)

type dnsConfig struct {
	vpcID  string
	debug  bool
	region string
}

func getDefaultRegion() string {
	val, present := os.LookupEnv(regionEnvVarStr)
	if present {
		return val
	} else {
		return regionDefault
	}
}
func NewCmdValidateDns() *cobra.Command {
	config := dnsConfig{}

	validateDnsCmd := &cobra.Command{
		Use: "dns",
		Run: func(cmd *cobra.Command, args []string) {
			// ctx
			ctx := context.TODO()

			// Create logger
			builder := ocmlog.NewStdLoggerBuilder()
			builder.Debug(config.debug)
			logger, err := builder.Build()
			if err != nil {
				fmt.Printf("Unable to build logger: %s\n", err.Error())
				os.Exit(1)
			}

			logger.Warn(ctx, "Using region: %s", config.region)
			creds := credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))
			// The use of t3.micro here is arbitrary; we just need to provide any valid machine type
			cli, err := cloudclient.NewClient(ctx, logger, creds, config.region, "t3.micro", nil)
			if err != nil {
				logger.Error(ctx, err.Error())
				os.Exit(1)
			}

			out := cli.VerifyDns(ctx, config.vpcID)
			out.Summary()
			if !out.IsSuccessful() {
				logger.Error(ctx, "Failure!")
				os.Exit(1)
			}

			logger.Info(ctx, "Success")
		},
	}

	validateDnsCmd.Flags().StringVar(&config.vpcID, "vpc-id", "", "ID of the VPC under test")
	validateDnsCmd.Flags().StringVar(&config.region, "region", getDefaultRegion(), fmt.Sprintf("Region to validate. Defaults to exported var %[1]v or '%[2]v' if not %[1]v set", regionEnvVarStr, regionDefault))
	validateDnsCmd.Flags().BoolVar(&config.debug, "debug", false, "If true, enable additional debug-level logging")

	if err := validateDnsCmd.MarkFlagRequired("vpc-id"); err != nil {
		validateDnsCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateDnsCmd

}
