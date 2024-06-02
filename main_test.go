package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ory/dockertest/v3"
)

var svc *s3.Client

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}
	runOptions := &dockertest.RunOptions{
		Repository: "quay.io/minio/minio",
		Cmd:        []string{"server", "/data"},
	}
	resource, err := pool.RunWithOptions(runOptions)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}()
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("http://localhost:%s", resource.GetPort("9000/tcp")),
		}, nil
	})
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion("auto"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				"minioadmin",
				"minioadmin",
				"",
			),
		),
	)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	svc = s3.NewFromConfig(cfg, func(op *s3.Options) {
		op.UsePathStyle = true
	})

	if err := pool.Retry(func() error {
		_, err := svc.CreateBucket(context.Background(), &s3.CreateBucketInput{
			Bucket: aws.String("my-bucket-2023e4a"),
		})
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Printf("Could not connect to database: %s", err)
		return 1
	}

	return m.Run()
}

func Test_upload(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	err := os.WriteFile(filePath, []byte("test-data-in-file"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = upload(ctx, svc, filePath)
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String("my-bucket-2023e4a"),
		Key:    aws.String("test"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer out.Body.Close()

	b, err := io.ReadAll(out.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "test-data-in-file" {
		t.Fatalf("want %s, got %s", "test-data-in-file", string(b))
	}
}
