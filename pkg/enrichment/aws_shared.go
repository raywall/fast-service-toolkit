package enrichment

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

var (
	awsCfg  aws.Config
	awsOnce sync.Once
	awsErr  error
)

// GetAWSConfig carrega a configuração da AWS (env vars, profile, IAM role) de forma lazy-singleton.
func GetAWSConfig(ctx context.Context, region string) (aws.Config, error) {
	awsOnce.Do(func() {
		opts := []func(*config.LoadOptions) error{}
		if region != "" {
			opts = append(opts, config.WithRegion(region))
		}
		awsCfg, awsErr = config.LoadDefaultConfig(ctx, opts...)
	})
	return awsCfg, awsErr
}
