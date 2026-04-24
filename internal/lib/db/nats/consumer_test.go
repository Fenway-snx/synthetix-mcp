package nats_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
)

func Test_MakeConsumer_ReplacingDotsWithDashes(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected string
	}{
		{
			name:     "one dot",
			data:     "thingo.new",
			expected: fmt.Sprintf("%s-%s", snx_lib_db_nats.DefaultPrefix, "thingo-new"),
		},
		{
			name:     "two dots",
			data:     "thingo..new",
			expected: fmt.Sprintf("%s-%s", snx_lib_db_nats.DefaultPrefix, "thingo--new"),
		},
		{
			name:     "complex one",
			data:     "new.sub.plus.another",
			expected: fmt.Sprintf("%s-%s", snx_lib_db_nats.DefaultPrefix, "new-sub-plus-another"),
		},
		{
			name:     "one trailing dash",
			data:     "new.sub.plus.another.",
			expected: fmt.Sprintf("%s-%s", snx_lib_db_nats.DefaultPrefix, "new-sub-plus-another"),
		},
		{
			name:     "multiple trailing dashes",
			data:     "new.sub.plus.another...",
			expected: fmt.Sprintf("%s-%s", snx_lib_db_nats.DefaultPrefix, "new-sub-plus-another"),
		},
		{
			name:     "one leading dash",
			data:     ".new.sub.plus.another",
			expected: fmt.Sprintf("%s-%s", snx_lib_db_nats.DefaultPrefix, "new-sub-plus-another"),
		},
		{
			name:     "multiple leading dashes",
			data:     "...new.sub.plus.another",
			expected: fmt.Sprintf("%s-%s", snx_lib_db_nats.DefaultPrefix, "new-sub-plus-another"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, snx_lib_db_nats.ConsumerName{Base: snx_lib_db_nats.ConsumerNameBase(test.data)}.String())
		})
	}
}

func Test_ConsumerName_RelayerPerSignerNames(t *testing.T) {
	prefix := snx_lib_db_nats.DefaultPrefix

	tests := []struct {
		name     string
		consumer snx_lib_db_nats.ConsumerName
		expected string
	}{
		{
			name:     "pending relayer consumer name",
			consumer: snx_lib_db_nats.ConsumerNameRelayerPendingTxRelayer,
			expected: fmt.Sprintf("%s-relayer-pending-tx-relayer-consumer", prefix),
		},
		{
			name:     "pending teller consumer name",
			consumer: snx_lib_db_nats.ConsumerNameRelayerPendingTxTeller,
			expected: fmt.Sprintf("%s-relayer-pending-tx-teller-consumer", prefix),
		},
		{
			name:     "pending watcher consumer name",
			consumer: snx_lib_db_nats.ConsumerNameRelayerPendingTxWatcher,
			expected: fmt.Sprintf("%s-relayer-pending-tx-watcher-consumer", prefix),
		},
		{
			name:     "relayer consumer name",
			consumer: snx_lib_db_nats.ConsumerNameRelayerTxnQueue,
			expected: fmt.Sprintf("%s-relayer-txn-queue-consumer", prefix),
		},
		{
			name:     "teller consumer name",
			consumer: snx_lib_db_nats.ConsumerNameRelayerTxnQueueTeller,
			expected: fmt.Sprintf("%s-relayer-txn-queue-teller-consumer", prefix),
		},
		{
			name:     "watcher consumer name",
			consumer: snx_lib_db_nats.ConsumerNameRelayerTxnQueueWatcher,
			expected: fmt.Sprintf("%s-relayer-txn-queue-watcher-consumer", prefix),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.consumer.String())
		})
	}
}

func assertRelayerSignerConsumerConfigDefaults(t *testing.T, cfg jetstream.ConsumerConfig) {
	t.Helper()
	assert.Equal(t, jetstream.AckExplicitPolicy, cfg.AckPolicy)
	assert.Equal(t, 7*time.Minute, cfg.AckWait)
	assert.Equal(t, 1, cfg.MaxDeliver)
	assert.Equal(t, jetstream.DeliverAllPolicy, cfg.DeliverPolicy)
	assert.Equal(t, jetstream.ReplayInstantPolicy, cfg.ReplayPolicy)
	assert.Equal(t, 1, cfg.MaxAckPending)
	assert.Equal(t, 1, cfg.MaxRequestBatch)
}

func Test_CreateRelayerTxnQueueRelayerConsumerConfig(t *testing.T) {
	cfg := snx_lib_db_nats.CreateRelayerTxnQueueRelayerConsumerConfig()

	assert.Equal(t, snx_lib_db_nats.ConsumerNameRelayerTxnQueue.String(), cfg.Durable)
	assertRelayerSignerConsumerConfigDefaults(t, cfg)

	assert.Empty(t, cfg.FilterSubject, "FilterSubject must not be set when FilterSubjects is used")
	assert.Len(t, cfg.FilterSubjects, 2)
	assert.Contains(t, cfg.FilterSubjects, snx_lib_db_nats.RelayerTxnQueue.String())
	assert.Contains(t, cfg.FilterSubjects, snx_lib_db_nats.RelayerTxnQueueRelayer.String())
}

func Test_CreateRelayerTxnQueueTellerConsumerConfig(t *testing.T) {
	cfg := snx_lib_db_nats.CreateRelayerTxnQueueTellerConsumerConfig()

	assert.Equal(t, snx_lib_db_nats.ConsumerNameRelayerTxnQueueTeller.String(), cfg.Durable)
	assert.Equal(t, snx_lib_db_nats.RelayerTxnQueueTeller.String(), cfg.FilterSubject)
	assert.Nil(t, cfg.FilterSubjects, "FilterSubjects must not be set when FilterSubject is used")
	assertRelayerSignerConsumerConfigDefaults(t, cfg)
}

func Test_CreateRelayerTxnQueueWatcherConsumerConfig(t *testing.T) {
	cfg := snx_lib_db_nats.CreateRelayerTxnQueueWatcherConsumerConfig()

	assert.Equal(t, snx_lib_db_nats.ConsumerNameRelayerTxnQueueWatcher.String(), cfg.Durable)
	assert.Equal(t, snx_lib_db_nats.RelayerTxnQueueWatcher.String(), cfg.FilterSubject)
	assert.Nil(t, cfg.FilterSubjects, "FilterSubjects must not be set when FilterSubject is used")
	assertRelayerSignerConsumerConfigDefaults(t, cfg)
}

func Test_CreateRelayerPendingTxConsumerConfig(t *testing.T) {
	consumerName := snx_lib_db_nats.ConsumerName{Base: "test-pending-consumer"}
	subject := snx_lib_db_nats.Subject{SubjectBase: "test.pending.subject"}
	cfg := snx_lib_db_nats.CreateRelayerPendingTxConsumerConfig(consumerName, subject)

	assert.Equal(t, consumerName.String(), cfg.Durable)
	assert.Equal(t, subject.String(), cfg.FilterSubject)
	assert.Equal(t, jetstream.AckExplicitPolicy, cfg.AckPolicy)
	assert.Equal(t, 10*time.Minute, cfg.AckWait)
	assert.Equal(t, snx_lib_db_nats.Unlimited, cfg.MaxDeliver)
	assert.Equal(t, jetstream.DeliverAllPolicy, cfg.DeliverPolicy)
	assert.Equal(t, jetstream.ReplayInstantPolicy, cfg.ReplayPolicy)
	assert.Equal(t, 1, cfg.MaxAckPending)
}

func Test_MakeConsumerConfigWithDefaults(t *testing.T) {
	cfg := snx_lib_db_nats.MakeConsumerConfigWithDefaults("test-consumer", "test.subject")

	assert.Equal(t, "test-consumer", cfg.Durable)
	assert.Equal(t, "test.subject", cfg.FilterSubject)
	assert.Equal(t, jetstream.AckExplicitPolicy, cfg.AckPolicy)
	assert.Equal(t, snx_lib_db_nats.AckWaitDefault, cfg.AckWait)
	assert.Equal(t, snx_lib_db_nats.Unlimited, cfg.MaxDeliver)
	assert.Equal(t, jetstream.ReplayInstantPolicy, cfg.ReplayPolicy)
	assert.Equal(t, snx_lib_db_nats.Unlimited, cfg.MaxAckPending)
}

func Test_MakeConsumerConfig_WithDeliverPolicy(t *testing.T) {
	cfg := snx_lib_db_nats.MakeConsumerConfig("test-consumer", "test.subject", jetstream.DeliverNewPolicy)

	assert.Equal(t, "test-consumer", cfg.Durable)
	assert.Equal(t, "test.subject", cfg.FilterSubject)
	assert.Equal(t, jetstream.DeliverNewPolicy, cfg.DeliverPolicy)
	assert.Equal(t, jetstream.AckExplicitPolicy, cfg.AckPolicy)
	assert.Equal(t, snx_lib_db_nats.AckWaitDefault, cfg.AckWait)
}
