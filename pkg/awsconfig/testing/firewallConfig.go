package main

import (
	"fmt"
	"os"
	"time"
	"flag"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/service/networkfirewall/networkfirewalliface"

	"k8s.io/apimachinery/pkg/util/wait"
)

type Client struct {
	ec2Client      ec2iface.EC2API
	firewallClient networkfirewalliface.NetworkFirewallAPI
}

type awsFlag struct {
	profile string
	region string
}

var wait2 = 360 * time.Second
var awsClient Client
var profile string
var region string

func main() {

    awsProfile := flag.String("p", "", "aws profile")
    awsRegion := flag.String("r", "", "aws region")
    flag.Parse()
	if (*awsProfile != ""){
		fmt.Println("Profile: ", *awsProfile)
		profile = *awsProfile
	}else{
		profile = os.Getenv("AWS_PROFILE")
	}
	if (*awsRegion != ""){
		fmt.Println("Region: ", *awsRegion)
		region = *awsRegion
	}else{
		region = os.Getenv("AWS_REGION")
	}

	if profile == ""{
		fmt.Println("Profile is not provided, will take in ENV")
		awsClient = EnvClient(region)
	}else{
		awsClient = ProfileClient(region, profile)
	}	
	//Create VPC
	Vpc := awsClient.CreateVPC()
	//Create Internet Gateway
	IG := awsClient.CreateInternetGatewayForVpc(*Vpc.Vpc.VpcId)
	//Create Public Subnet
	PublicSubnet := awsClient.CreateSubnet("10.0.0.0/24", *Vpc.Vpc.VpcId)
	//Create Private Subnet
	PrivateSubnet := awsClient.CreateSubnet("10.0.1.0/24", *Vpc.Vpc.VpcId)
	//Create Firewall Subnet
	FirewallSubnet := awsClient.CreateSubnet("10.0.2.0/24", *Vpc.Vpc.VpcId)
	//Create PublicSubnet Route Table
	PublicRT := awsClient.CreateRouteTableForSubnet(*Vpc.Vpc.VpcId,*PublicSubnet.Subnet.SubnetId)
	//Create PrivateSubnet Route Table
	PrivateRT := awsClient.CreateRouteTableForSubnet(*Vpc.Vpc.VpcId,*PrivateSubnet.Subnet.SubnetId)
	//Create FirewallSubnet Route Table
	FirewallRT := awsClient.CreateRouteTableForSubnet(*Vpc.Vpc.VpcId,*FirewallSubnet.Subnet.SubnetId)
	//Create IGW Route Table
	IgRT := awsClient.CreateRouteTableForIGW(*Vpc.Vpc.VpcId, *IG.InternetGateway.InternetGatewayId)
	//Create NAT Gateway
	NatGateway := awsClient.CreateNatGateway(*PublicSubnet.Subnet.SubnetId)

	//Create route 0.0.0.0/0 in PrivateRT for NatGateway
	awsClient.CreateRouteForGateway("0.0.0.0/0", *NatGateway.NatGateway.NatGatewayId, *PrivateRT.RouteTable.RouteTableId)
	fmt.Println("Successfully Created a route 0.0.0.0/0 to NatGateway in Private Subnet")
	//Create route 0.0.0.0/0 in FirewallSubnet for IG
	awsClient.CreateRouteForGateway("0.0.0.0/0", *IG.InternetGateway.InternetGatewayId, *FirewallRT.RouteTable.RouteTableId)
	fmt.Println("Successfully Created a route 0.0.0.0/0 to IGW in Firewall Subnet")

	//Create Firewall
	Firewall := awsClient.CreateFirewall(FirewallSubnet, Vpc)
	//Wait for 6 minutes for the Firewall to be ready
	fmt.Println("It's gonna take around 6 minutes for the Firewall Vpc Endpoint to become available")
	if awsClient.IsFirewallReady(Firewall) != nil {
		fmt.Println(awsClient.IsFirewallReady(Firewall).Error())
	}
	fmt.Println("VpcEndpoint is Now Available!")

	DescribeFirewall := awsClient.DescribeFirewall(Firewall)
	//Create route 0.0.0.0/0 in PublicRT for FirewallEndpoint
	firewallEndpointId := *DescribeFirewall.FirewallStatus.SyncStates[*FirewallSubnet.Subnet.AvailabilityZone].Attachment.EndpointId
	//Check to see if the Firewall VpcEndpoint is available
	awsClient.CreateRouteToFirewall("0.0.0.0/0", firewallEndpointId, PublicRT)
	fmt.Println("Successfully route 0.0.0.0/0 to the Firewall Endpoint in PublicRT ")
	//Create route 10.0.0.0/24 in IgRt to FirewallEndpoint
	awsClient.CreateRouteToFirewall("10.0.0.0/24", firewallEndpointId, IgRT)
	fmt.Println("Successfully route 10.0.0.0/24 to the Firewall Endpoint in IgRT ")

}

func EnvClient(region string) Client{
	creds := credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))

	ec2Client := ec2.New(session.New(&aws.Config{
		Region:      &region,
		Credentials: creds,
	}))

	firewallClient := networkfirewall.New(session.Must(session.NewSession()), &aws.Config{
		Region:      &region,
		Credentials: creds,
	})

	awsClient := Client{
		ec2Client,
		firewallClient,
	}
	return awsClient
}

func ProfileClient(region, profile string) Client {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:      &region,
		},
		Profile: profile,
	}))
	if _, err := sess.Config.Credentials.Get(); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NoCredentialProviders":
				fmt.Println("could not create AWS session: ", err)
			default:
				fmt.Println("could not create AWS session: ", err)
			}
		}
	}

	ec2Client := ec2.New(sess)
	firewallClient := networkfirewall.New(sess)

	awsClient := Client{
		ec2Client,
		firewallClient,
	}
	return awsClient
}

func (c Client) CreateVPC() ec2.CreateVpcOutput {
	VPC, err := c.ec2Client.CreateVpc(&ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
				default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	fmt.Println("Successfully created a vpc with ID:", string(*VPC.Vpc.VpcId))
	//Enable DNSHostname
	_, err = c.ec2Client.ModifyVpcAttribute(&ec2.ModifyVpcAttributeInput{
		EnableDnsHostnames: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
		VpcId: aws.String(*VPC.Vpc.VpcId),
	})
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("Successfully enabled DNSHostname for the newly created VPC")
	return *VPC
}

func (c Client) CreateInternetGatewayForVpc(vpcID string) ec2.CreateInternetGatewayOutput {
	IGresult, err := c.ec2Client.CreateInternetGateway(&ec2.CreateInternetGatewayInput{})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	fmt.Println("Successfully created IG")
	//Attach the InternetGateway to the VPC
	_, err = c.ec2Client.AttachInternetGateway(&ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(*IGresult.InternetGateway.InternetGatewayId),
		VpcId:             aws.String(vpcID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	fmt.Println("Successfully attached IG to VPC")

	return *IGresult
}

func (c Client) CreateSubnet(CidrBlock string, vpcID string) ec2.CreateSubnetOutput {
	
	Subnet, err := c.ec2Client.CreateSubnet(&ec2.CreateSubnetInput{
		CidrBlock: aws.String(CidrBlock),
		VpcId:     aws.String(vpcID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	return *Subnet
}

func (c Client) CreateRouteTableForSubnet(vpcID string, subnetID string) ec2.CreateRouteTableOutput {
	RT, err := c.ec2Client.CreateRouteTable(&ec2.CreateRouteTableInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	Associateinput := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*RT.RouteTable.RouteTableId),
		SubnetId:     aws.String(subnetID),
	}
	_, err = c.ec2Client.AssociateRouteTable(Associateinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	return *RT
}

func (c Client) CreateRouteTableForIGW(vpcId string, IgwId string) ec2.CreateRouteTableOutput {
	RouteTable1input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(vpcId),
	}

	RT, err := c.ec2Client.CreateRouteTable(RouteTable1input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	Associateinput := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*RT.RouteTable.RouteTableId),
		GatewayId:    aws.String(IgwId),
	}
	_, err = c.ec2Client.AssociateRouteTable(Associateinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	return *RT
}
func (c Client) CreateNatGateway(SubnetId string) ec2.CreateNatGatewayOutput {
	EIPinput := &ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	}

	EIPresult, err := c.ec2Client.AllocateAddress(EIPinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	NGinput := &ec2.CreateNatGatewayInput{
		AllocationId: aws.String(*EIPresult.AllocationId),
		SubnetId:     aws.String(SubnetId),
	}

	NGresult, err := c.ec2Client.CreateNatGateway(NGinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	fmt.Println("Waiting 2 minutes for NAT Gateway to become ready")
	//Wait for NAT Gatway to be ready
	err = c.ec2Client.WaitUntilNatGatewayAvailable(&ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []*string{aws.String(*NGresult.NatGateway.NatGatewayId)},
	})
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("Successfully create a NAT Gateway")
	return *NGresult
}
func (c Client) CreateRouteForGateway(CidrBlock string, GatewayID string, RouteTableId string ) {
	Ruleinput := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String(CidrBlock),
		GatewayId:            aws.String(GatewayID),
		RouteTableId:         aws.String(RouteTableId),
	}

	_, err := c.ec2Client.CreateRoute(Ruleinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

}
func (c Client) CreateFirewall(FirewallSubnet ec2.CreateSubnetOutput, Vpc ec2.CreateVpcOutput) networkfirewall.CreateFirewallOutput {
	RuleGroupinput := &networkfirewall.CreateRuleGroupInput{
		Capacity:      aws.Int64(100),
		RuleGroupName: aws.String("test-firewall"),
		Type:          aws.String("STATEFUL"),
		RuleGroup: &networkfirewall.RuleGroup{
			RulesSource: &networkfirewall.RulesSource{
				RulesSourceList: &networkfirewall.RulesSourceList{
					GeneratedRulesType: aws.String("DENYLIST"),
					TargetTypes:        []*string{aws.String("TLS_SNI")},
					Targets:            []*string{aws.String(".quay.io"), aws.String("api.openshift.com"), aws.String(".redhat.io")},
				},
			},
		},
	}

	statefulRuleGroup, err := c.firewallClient.CreateRuleGroup(RuleGroupinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	fmt.Println("Successfully creates a Stateful Rule Group")

	FirewallPolicyInput := &networkfirewall.CreateFirewallPolicyInput{
		Description:        aws.String("test"),
		FirewallPolicyName: aws.String("testPolicy"),
		FirewallPolicy: &networkfirewall.FirewallPolicy{
			StatefulRuleGroupReferences: []*networkfirewall.StatefulRuleGroupReference{&networkfirewall.StatefulRuleGroupReference{
				ResourceArn: statefulRuleGroup.RuleGroupResponse.RuleGroupArn},
			},
			StatelessDefaultActions:         []*string{aws.String("aws:forward_to_sfe")},
			StatelessFragmentDefaultActions: []*string{aws.String("aws:forward_to_sfe")},
		},
	}
	testFirewallPolicy, err := c.firewallClient.CreateFirewallPolicy(FirewallPolicyInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	fmt.Println("Successfully created a Firewall Policy")

	testFirewallInput := &networkfirewall.CreateFirewallInput{
		FirewallName:      aws.String("testFirewall"),
		FirewallPolicyArn: testFirewallPolicy.FirewallPolicyResponse.FirewallPolicyArn,
		SubnetMappings: []*networkfirewall.SubnetMapping{&networkfirewall.SubnetMapping{
			SubnetId: FirewallSubnet.Subnet.SubnetId},
		},
		VpcId: aws.String(*Vpc.Vpc.VpcId),
	}

	Firewall, err := c.firewallClient.CreateFirewall(testFirewallInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	fmt.Println("Successfully created a Firewall!")
	return *Firewall

}
func (c Client) DescribeFirewall(Firewall networkfirewall.CreateFirewallOutput) networkfirewall.DescribeFirewallOutput {
	DescribeFirewall, err := c.firewallClient.DescribeFirewall(&networkfirewall.DescribeFirewallInput{
		FirewallName: aws.String(*Firewall.Firewall.FirewallName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	return *DescribeFirewall
}

func (c Client) IsFirewallReady(Firewall networkfirewall.CreateFirewallOutput) error {
	DescribeFirewall, err := c.firewallClient.DescribeFirewall(&networkfirewall.DescribeFirewallInput{
		FirewallName: aws.String(*Firewall.Firewall.FirewallName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	FirewallStatus := func() (bool, error) {
		if DescribeFirewall.FirewallStatus.Status == aws.String("READY") {
			return true, nil
		}
		return false, nil
	}
	err = wait.PollImmediate(2*time.Second, wait2, FirewallStatus)
	fmt.Println("Polling in Action: checking Firewall Status", err)
	return err
}

func (c Client) CreateRouteToFirewall(CidrBlock string, VPCEndpointId string, RouteTable ec2.CreateRouteTableOutput) {
	Ruleinput := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String(CidrBlock),
		VpcEndpointId:        aws.String(VPCEndpointId),
		RouteTableId:         aws.String(*RouteTable.RouteTable.RouteTableId),
	}

	_, err := c.ec2Client.CreateRoute(Ruleinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

}
