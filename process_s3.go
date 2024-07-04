package main

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func downloadBackupFromS3(ctx context.Context) (string, error) {
	log.Info().Msg("Downloading SQL file from S3...")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"), // Update with your region
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create AWS session: %v", err)
	}

	s3Client := s3.New(sess)

	tempFile, err := ioutil.TempFile("", "backup-*.sql")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer tempFile.Close()

	resp, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3BucketName),
		Key:    aws.String(s3FileKey),
	})
	if err != nil {
		return "", fmt.Errorf("failed to download file from S3: %v", err)
	}
	defer resp.Body.Close()

	_, err = tempFile.ReadFrom(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read from S3 response body: %v", err)
	}

	_ = restoreBackupToDB(ctx, tempFile.Name())
	return tempFile.Name(), nil
}

func restoreBackupToDB(ctx context.Context, filePath string) error {

	backupFile, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %v", err)
	}

	_, err = dbPool.Exec(ctx, string(backupFile))
	if err != nil {
		return fmt.Errorf("failed to execute backup SQL: %v", err)
	}

	log.Info().Msg("Restored DB from SQL")
	return nil
}
