package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	ec2svc "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/containernetworking/plugins/pkg/utils"
)

const (
	maxRetries  = 5
	handlerName = "multi-nic-aws"
)

var (
	httpTimeoutValue = 10 * time.Second
)

// EC2 is the EC2 wrapper interface
type EC2 interface {
	DescribeNetworkInterfacesWithContext(ctx aws.Context, input *ec2svc.DescribeNetworkInterfacesInput, opts ...request.Option) (*ec2svc.DescribeNetworkInterfacesOutput, error)

	AssignPrivateIpAddressesWithContext(ctx aws.Context, input *ec2svc.AssignPrivateIpAddressesInput, opts ...request.Option) (*ec2svc.AssignPrivateIpAddressesOutput, error)
	UnassignPrivateIpAddressesWithContext(ctx aws.Context, input *ec2svc.UnassignPrivateIpAddressesInput, opts ...request.Option) (*ec2svc.UnassignPrivateIpAddressesOutput, error)
}

func getClient() (EC2, error) {
	sess := session.New()
	ec2Metadata := ec2metadata.New(sess)
	region, err := ec2Metadata.Region()
	if err != nil {
		return nil, fmt.Errorf("Failed to get region: %v", err)
	}

	cfg := aws.NewConfig().WithRegion(region)
	sess = sess.Copy(cfg)
	client := ec2svc.New(sess)
	return client, nil
}

func getENIId(client EC2, primaryAddress string) (*string, error) {
	input := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("addresses.private-ip-address"),
				Values: []*string{aws.String(primaryAddress)},
			},
		},
	}
	output, err := client.DescribeNetworkInterfacesWithContext(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("Failed to DescribeNetworkInterfaces: %v", err)
	}
	if len(output.NetworkInterfaces) == 0 {
		return nil, fmt.Errorf("Cannot getENIId: no interface found")
	}
	eniID := output.NetworkInterfaces[0].NetworkInterfaceId
	return eniID, err
}

func AssignIP(primaryAddress, podIP string) (*ec2svc.AssignPrivateIpAddressesOutput, error) {
	client, err := getClient()
	if client == nil {
		return nil, err
	}
	eniId, err := getENIId(client, primaryAddress)
	if err != nil {
		return nil, err
	}
	input := &ec2.AssignPrivateIpAddressesInput{
		AllowReassignment:  aws.Bool(true),
		NetworkInterfaceId: eniId,
		PrivateIpAddresses: []*string{
			aws.String(podIP),
		},
	}
	utils.Logger.Debug(fmt.Sprintf("AssignPrivateIpAddresses %s to %s", podIP, *eniId))
	output, err := client.AssignPrivateIpAddressesWithContext(context.Background(), input)
	return output, err
}

func UnassignIP(primaryAddress, podIP string) error {
	client, err := getClient()
	if client == nil {
		return err
	}
	eniId, err := getENIId(client, primaryAddress)
	if err != nil {
		return err
	}
	input := &ec2.UnassignPrivateIpAddressesInput{
		NetworkInterfaceId: eniId,
		PrivateIpAddresses: []*string{
			aws.String(podIP),
		},
	}
	utils.Logger.Debug(fmt.Sprintf("UnassignPrivateIpAddresses %s from %s", podIP, *eniId))
	_, err = client.UnassignPrivateIpAddressesWithContext(context.Background(), input)
	return err
}
