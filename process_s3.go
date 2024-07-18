package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"os"
	"os/exec"
)

func downloadBackupFromS3() (string, error) {
	log.Info().Msg("Downloading SQL file from S3...")
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"), // Update with your region
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create AWS session: %v", err)
	}

	s3Client := s3.New(sess)

	tempFile, err := ioutil.TempFile("", "backup-*.sql.gz")
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
	fileName := tempFile.Name()

	err = gunzipFile(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to extract file: %v", err)
	}
	_ = restoreBackupToDB(fileName[:len(fileName)-3])
	return tempFile.Name(), nil
}

func gunzipFile(src string) error {
	cmd := exec.Command("gunzip", src)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run gunzip command: %v", err)
	}

	log.Info().Msg("SQL file extracted.")
	return nil
}

func restoreBackupToDB(filePath string) error {

	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		dbPool.Config().ConnConfig.Host, dbPool.Config().ConnConfig.Port,
		dbPool.Config().ConnConfig.Database, dbPool.Config().ConnConfig.User, dbPool.Config().ConnConfig.Password)

	cmd := exec.Command("psql", connStr)
	cmd.Stdin, _ = os.Open(filePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Err(err).Msg("Failed to run psql command")
		return err
	}

	log.Info().Msg("Restored DB from SQL")
	return nil
}
