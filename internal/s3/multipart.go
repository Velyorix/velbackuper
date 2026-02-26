package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (c *Client) UploadMultipart(ctx context.Context, key string, body io.Reader, partSizeBytes int64) error {
	if partSizeBytes < MinPartSizeBytes {
		partSizeBytes = MinPartSizeBytes
	}
	fullKey := c.Key(key)

	createOut, err := c.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return fmt.Errorf("create multipart upload: %w", err)
	}
	uploadID := createOut.UploadId
	defer func() {
		if uploadID != nil {
			_, _ = c.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
				Bucket:   aws.String(c.bucket),
				Key:      aws.String(fullKey),
				UploadId: uploadID,
			})
		}
	}()

	var completed []types.CompletedPart
	partNumber := int32(1)
	buf := make([]byte, partSizeBytes)

	for {
		n, readErr := io.ReadFull(body, buf)
		if n == 0 && readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return fmt.Errorf("read part: %w", readErr)
		}
		if n == 0 {
			break
		}

		partBody := bytes.NewReader(buf[:n])
		uploadOut, err := c.client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:        aws.String(c.bucket),
			Key:           aws.String(fullKey),
			UploadId:      uploadID,
			PartNumber:    aws.Int32(partNumber),
			Body:          partBody,
			ContentLength: aws.Int64(int64(n)),
		})
		if err != nil {
			return fmt.Errorf("upload part %d: %w", partNumber, err)
		}
		completed = append(completed, types.CompletedPart{
			ETag:       uploadOut.ETag,
			PartNumber: aws.Int32(partNumber),
		})
		partNumber++

		if readErr == io.EOF || (readErr == io.ErrUnexpectedEOF && n < len(buf)) {
			break
		}
	}

	if len(completed) == 0 {
		return fmt.Errorf("no parts uploaded")
	}

	_, err = c.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(fullKey),
		UploadId: uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completed,
		},
	})
	if err != nil {
		return fmt.Errorf("complete multipart upload: %w", err)
	}
	uploadID = nil
	return nil
}
