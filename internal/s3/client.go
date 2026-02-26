package s3

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	MinPartSizeMB    = 5
	MinPartSizeBytes = MinPartSizeMB * 1024 * 1024
)

type Options struct {
	Endpoint           string
	Region             string
	AccessKey          string
	SecretKey          string
	Bucket             string
	Prefix             string
	InsecureSkipVerify bool
}

type Client struct {
	client *s3.Client
	bucket string
	prefix string
}

func New(ctx context.Context, opts Options) (*Client, error) {
	if opts.Region == "" {
		opts.Region = "us-east-1"
	}
	endpointURL, err := url.Parse(strings.TrimSpace(opts.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("s3 endpoint: %w", err)
	}
	if endpointURL.Scheme == "" {
		endpointURL.Scheme = "https"
		endpointURL, _ = url.Parse(endpointURL.String())
	}

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpointURL.String(),
			SigningRegion:     opts.Region,
			HostnameImmutable: true,
		}, nil
	})

	cfg := aws.Config{
		Region:                      opts.Region,
		EndpointResolverWithOptions: resolver,
		Credentials:                 credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, ""),
	}

	httpClient := http.DefaultClient
	if opts.InsecureSkipVerify {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.HTTPClient = httpClient
	})

	return &Client{
		client: client,
		bucket: opts.Bucket,
		prefix: strings.Trim(opts.Prefix, "/"),
	}, nil
}

func (c *Client) Key(relative string) string {
	relative = strings.Trim(relative, "/")
	if c.prefix == "" {
		return relative
	}
	return path.Join(c.prefix, relative)
}

func (c *Client) Bucket() string {
	return c.bucket
}

func (c *Client) Prefix() string {
	return c.prefix
}

func (c *Client) PutObject(ctx context.Context, key string, body io.Reader, contentLength int64) error {
	fullKey := c.Key(key)
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(fullKey),
		Body:          body,
		ContentLength: aws.Int64(contentLength),
	})
	return err
}

func (c *Client) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	fullKey := c.Key(key)
	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (c *Client) DeleteObject(ctx context.Context, key string) error {
	fullKey := c.Key(key)
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fullKey),
	})
	return err
}

func (c *Client) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]string, error) {
	fullPrefix := c.Key(prefix)
	if fullPrefix != "" && !strings.HasSuffix(fullPrefix, "/") {
		fullPrefix += "/"
	}
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(c.bucket),
		Prefix:  aws.String(fullPrefix),
		MaxKeys: aws.Int32(maxKeys),
	}
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(c.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
		if maxKeys > 0 && int32(len(keys)) >= maxKeys {
			break
		}
	}
	return keys, nil
}

func (c *Client) Client() *s3.Client {
	return c.client
}
