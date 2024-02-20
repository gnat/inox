package cloudflareprovider

import (
	"strings"
	"testing"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/inoxlang/inox/internal/core"
	"github.com/inoxlang/inox/internal/globals/s3_ns"
	"github.com/stretchr/testify/assert"
)

var (
	objectStorageLimit = core.Limit{
		Name:  s3_ns.OBJECT_STORAGE_REQUEST_RATE_LIMIT_NAME,
		Kind:  core.FrequencyLimit,
		Value: 100 * core.FREQ_LIMIT_SCALE,
	}
)

func TestCreateS3CredentialsForSingleBucket(t *testing.T) {
	projectId := core.ProjectID("test-s3-creds-single-bucket")

	if cloudflareConfig.AdditionalTokensApiToken == "" {
		t.Skip()
		return
	}

	ctx := core.NewContextWithEmptyState(core.ContextConfig{
		Limits: []core.Limit{objectStorageLimit},
	}, nil)
	defer ctx.CancelGracefully()

	cf, err := New(projectId, &cloudflareConfig)
	if !assert.NoError(t, err) {
		return
	}

	//cleanup
	if !assert.NoError(t, cf.deleteHighPermsTokens(ctx)) {
		return
	}

	tokens, err := cf.GetUpToDateTempTokens(ctx)
	if !assert.NoError(t, err) {
		return
	}

	defer deleteTestRelatedTokens(t, ctx, cf.apiTokensApi, projectId)

	deleteBucket := func() {
		api, err := cloudflare.NewWithAPIToken(tokens.R2Token.Value)
		if !assert.NoError(t, err) {
			assert.Fail(t, err.Error())
			return
		}
		api.DeleteR2Bucket(ctx, cloudflare.AccountIdentifier(cloudflareConfig.AccountID), "temp")
	}
	deleteBucket()

	//tear down
	defer deleteBucket()

	creds, err := cf.GetCreateS3CredentialsForSingleBucket(ctx, "temp", projectId)

	if !assert.NoError(t, err) {
		return
	}

	//check that credentials work
	bucket, err := s3_ns.OpenBucketWithCredentials(ctx, s3_ns.OpenBucketWithCredentialsInput{
		S3Host:     "s3://temp",
		Provider:   "cloudflare",
		BucketName: "temp",
		HttpsHost:  creds.s3Endpoint,
		AccessKey:  creds.accessKey,
		SecretKey:  creds.secretKey,
	})

	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		bucket.RemoveAllObjects(ctx)
	}()

	_, err = bucket.PutObject(ctx, "x", strings.NewReader("content"))

	if !assert.NoError(t, err) {
		return
	}

	resp, err := bucket.GetObject(ctx, "x")
	if !assert.NoError(t, err) {
		return
	}

	p, err := resp.ReadAll()

	if !assert.NoError(t, err) {
		return
	}

	if !assert.Equal(t, "content", string(p)) {
		return
	}

	//the same credentials should be returned if asked again

	creds2, err := cf.GetCreateS3CredentialsForSingleBucket(ctx, "temp", projectId)

	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, creds, creds2)
}
