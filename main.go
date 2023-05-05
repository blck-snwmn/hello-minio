package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var accountID = os.Getenv("CF_ACCOUNT_ID")

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run main.go /path/to/local/file")
		return
	}
	//
	// コンフィグをロードする
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
		}, nil
	})
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("auto"), config.WithEndpointResolverWithOptions(customResolver))
	if err != nil {
		fmt.Println("failed to load config,", err)
		return
	}

	err = upload(cfg, os.Args[1])
	if err != nil {
		fmt.Println("failed to upload,", err)
		return
	}
	fmt.Println("success upload")
}

func upload(cfg aws.Config, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	svc := s3.NewFromConfig(cfg)
	_, err = svc.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String("my-bucket-2023e4a"),
		Key:    aws.String("test"),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}
