package logrpcs

import (
	"context"
	"log"
	"webapi_go/app"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3 uploads
var (
	// s3.NewFrom
	s3client *s3.Client
)

func init() {
	app.OnStart(func() {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		s3client = s3.NewFromConfig(cfg)
	})
}
