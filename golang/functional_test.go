package gubernator_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mailgun/gubernator/golang"
	"github.com/mailgun/gubernator/golang/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Setup and shutdown the mailgun mock server for the entire test suite
func TestMain(m *testing.M) {
	if err := cluster.Start(5); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer cluster.Stop()
	os.Exit(m.Run())
}

func TestOverTheLimit(t *testing.T) {
	client, errs := gubernator.NewClient(cluster.GetPeer())
	require.Nil(t, errs)

	tests := []struct {
		Remaining int64
		Status    gubernator.Status
	}{
		{
			Remaining: 1,
			Status:    gubernator.UnderLimit,
		},
		{
			Remaining: 0,
			Status:    gubernator.UnderLimit,
		},
		{
			Remaining: 0,
			Status:    gubernator.OverLimit,
		},
	}

	for _, test := range tests {
		resp, err := client.GetRateLimit(context.Background(), &gubernator.Request{
			Namespace: "test_over_limit",
			UniqueKey: "account:1234",
			Algorithm: gubernator.TokenBucket,
			Duration:  time.Second * 1,
			Limit:     2,
			Hits:      1,
		})
		require.Nil(t, err)

		assert.Equal(t, test.Status, resp.Status)
		assert.Equal(t, test.Remaining, resp.LimitRemaining)
		assert.Equal(t, int64(2), resp.CurrentLimit)
		assert.False(t, resp.ResetTime.IsZero())
	}
}

func TestTokenBucket(t *testing.T) {
	client, errs := gubernator.NewClient(cluster.GetPeer())
	require.Nil(t, errs)

	tests := []struct {
		Remaining int64
		Status    gubernator.Status
		Sleep     time.Duration
	}{
		{
			Remaining: 1,
			Status:    gubernator.UnderLimit,
			Sleep:     time.Duration(0),
		},
		{
			Remaining: 0,
			Status:    gubernator.UnderLimit,
			Sleep:     time.Duration(time.Millisecond * 5),
		},
		{
			Remaining: 1,
			Status:    gubernator.UnderLimit,
			Sleep:     time.Duration(0),
		},
	}

	for _, test := range tests {
		resp, err := client.GetRateLimit(context.Background(), &gubernator.Request{
			Namespace: "test_token_bucket",
			UniqueKey: "account:1234",
			Algorithm: gubernator.TokenBucket,
			Duration:  time.Millisecond * 5,
			Limit:     2,
			Hits:      1,
		})
		require.Nil(t, err)

		assert.Equal(t, test.Status, resp.Status)
		assert.Equal(t, test.Remaining, resp.LimitRemaining)
		assert.Equal(t, int64(2), resp.CurrentLimit)
		assert.False(t, resp.ResetTime.IsZero())
		time.Sleep(test.Sleep)
	}
}

func TestLeakyBucket(t *testing.T) {
	client, errs := gubernator.NewClient(cluster.GetPeer())
	require.Nil(t, errs)

	tests := []struct {
		Hits      int64
		Remaining int64
		Status    gubernator.Status
		Sleep     time.Duration
	}{
		{
			Hits:      5,
			Remaining: 0,
			Status:    gubernator.UnderLimit,
			Sleep:     time.Duration(0),
		},
		{
			Hits:      1,
			Remaining: 0,
			Status:    gubernator.OverLimit,
			Sleep:     time.Duration(time.Millisecond * 10),
		},
		{
			Hits:      1,
			Remaining: 0,
			Status:    gubernator.UnderLimit,
			Sleep:     time.Duration(time.Millisecond * 20),
		},
		{
			Hits:      1,
			Remaining: 1,
			Status:    gubernator.UnderLimit,
			Sleep:     time.Duration(0),
		},
	}

	for _, test := range tests {
		resp, err := client.GetRateLimit(context.Background(), &gubernator.Request{
			Namespace: "test_leaky_bucket",
			UniqueKey: "account:1234",
			Algorithm: gubernator.LeakyBucket,
			Duration:  time.Millisecond * 50,
			Hits:      test.Hits,
			Limit:     5,
		})
		require.Nil(t, err)

		assert.Equal(t, test.Status, resp.Status)
		assert.Equal(t, test.Remaining, resp.LimitRemaining)
		assert.Equal(t, int64(5), resp.CurrentLimit)
		assert.False(t, resp.ResetTime.IsZero())
		time.Sleep(test.Sleep)
	}
}