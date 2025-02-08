package main

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	// "crypto/x509"
	// "encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
)

func GeneratePublicSSHKey() (string, error) {
	// Generate a new RSA private key with 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Validate the private key
	if err := privateKey.Validate(); err != nil {
		return "", fmt.Errorf("failed to validate RSA key: %w", err)
	}

	// Generate the public key from the private key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate public key: %w", err)
	}

	// Encode the public key in OpenSSH format
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	return string(publicKeyBytes), nil
}

func main() {
	publicKey, err := GeneratePublicSSHKey()
	if err != nil {
		fmt.Printf("Error generating public SSH key: %v\n", err)
		return
	}
	pulumi.Run(func(ctx *pulumi.Context) error {
		blockchain_vpc, err := ec2.NewVpc(ctx, "blockchain", &ec2.VpcArgs{
			CidrBlock: pulumi.String("10.0.0.0/16"),
		})
		if err != nil {
			return err
		}
		// PUBLIC SUBNET
		public_snet, err := ec2.NewSubnet(ctx, "public-snet", &ec2.SubnetArgs{
			VpcId:     blockchain_vpc.ID(),
			CidrBlock: pulumi.String("10.0.1.0/24"),
		})
		if err != nil {
			return err
		}
		ctx.Export("private_snet", public_snet.ID())

		// SECURITY GROUP
		security_group, err := ec2.NewSecurityGroup(ctx, "sec-group", &ec2.SecurityGroupArgs{
			VpcId: blockchain_vpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		})
		if err != nil {
			return err
		}

		key_pair, err := ec2.NewKeyPair(ctx, "ec2-public-key", &ec2.KeyPairArgs{
			KeyName:   pulumi.String("blockchain-key"),
			PublicKey: pulumi.String(publicKey),
		})

		if err != nil {
			return err
		}

		instance, err := ec2.NewInstance(ctx, "blockchain-server", &ec2.InstanceArgs{
			Ami:                      pulumi.String("ami-085ad6ae776d8f09c"),
			SubnetId:                 public_snet.ID(),
			InstanceType:             pulumi.String("t2.micro"),
			KeyName:                  key_pair.ID(),
			AssociatePublicIpAddress: pulumi.Bool(true),
			SecurityGroups: pulumi.StringArray{
				security_group.ID(),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("instace ip", instance.PublicIp)
		ctx.Export("ssh key", key_pair.PublicKey)

		return nil

	})
}
