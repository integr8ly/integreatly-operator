// a helper utility for the cluster network provider.
// the network provider provisions subnets in two availability zones.
// to ensure a mapping between subnets and availability zones this sort utility,
// allows for deterministic sorting of availability zones based on the zone names
package aws

import "github.com/aws/aws-sdk-go/service/ec2"

type azByZoneName []*ec2.AvailabilityZone

func (z azByZoneName) Len() int           { return len(z) }
func (z azByZoneName) Less(i, j int) bool { return *z[i].ZoneName < *z[j].ZoneName }
func (z azByZoneName) Swap(i, j int)      { z[i], z[j] = z[j], z[i] }
