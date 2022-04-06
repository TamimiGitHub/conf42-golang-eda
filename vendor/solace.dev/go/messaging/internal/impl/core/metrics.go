// pubsubplus-go-client
//
// Copyright 2021-2022 Solace Corporation. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"fmt"
	"sync"
	"sync/atomic"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/pkg/solace/metrics"
)

var rxMetrics = map[metrics.Metric]ccsmp.SolClientStatsRX{
	metrics.DirectBytesReceived:                       ccsmp.SolClientStatsRXDirectBytes,
	metrics.DirectMessagesReceived:                    ccsmp.SolClientStatsRXDirectMsgs,
	metrics.BrokerDiscardNotificationsReceived:        ccsmp.SolClientStatsRXDiscardInd,
	metrics.UnknownParameterMessagesDiscarded:         ccsmp.SolClientStatsRXDiscardSmfUnknownElement,
	metrics.TooBigMessagesDiscarded:                   ccsmp.SolClientStatsRXDiscardMsgTooBig,
	metrics.PersistentAcknowledgeSent:                 ccsmp.SolClientStatsRXAcked,
	metrics.PersistentDuplicateMessagesDiscarded:      ccsmp.SolClientStatsRXDiscardDuplicate,
	metrics.PersistentNoMatchingFlowMessagesDiscarded: ccsmp.SolClientStatsRXDiscardNoMatchingFlow,
	metrics.PersistentOutOfOrderMessagesDiscarded:     ccsmp.SolClientStatsRXDiscardOutoforder,
	metrics.PersistentBytesReceived:                   ccsmp.SolClientStatsRXPersistentBytes,
	metrics.PersistentMessagesReceived:                ccsmp.SolClientStatsRXPersistentMsgs,
	metrics.ControlMessagesReceived:                   ccsmp.SolClientStatsRXCtlMsgs,
	metrics.ControlBytesReceived:                      ccsmp.SolClientStatsRXCtlBytes,
	metrics.TotalBytesReceived:                        ccsmp.SolClientStatsRXTotalDataBytes,
	metrics.TotalMessagesReceived:                     ccsmp.SolClientStatsRXTotalDataMsgs,
	metrics.CompressedBytesReceived:                   ccsmp.SolClientStatsRXCompressedBytes,
}

var txMetrics = map[metrics.Metric]ccsmp.SolClientStatsTX{
	metrics.TotalBytesSent:                   ccsmp.SolClientStatsTXTotalDataBytes,
	metrics.TotalMessagesSent:                ccsmp.SolClientStatsTXTotalDataMsgs,
	metrics.DirectBytesSent:                  ccsmp.SolClientStatsTXDirectBytes,
	metrics.DirectMessagesSent:               ccsmp.SolClientStatsTXDirectMsgs,
	metrics.PersistentBytesSent:              ccsmp.SolClientStatsTXPersistentBytes,
	metrics.PersistentMessagesSent:           ccsmp.SolClientStatsTXPersistentMsgs,
	metrics.PersistentBytesRedelivered:       ccsmp.SolClientStatsTXPersistentBytesRedelivered,
	metrics.PersistentMessagesRedelivered:    ccsmp.SolClientStatsTXPersistentRedelivered,
	metrics.PublisherAcknowledgementReceived: ccsmp.SolClientStatsTXAcksRxed,
	metrics.PublisherWindowClosed:            ccsmp.SolClientStatsTXWindowClose,
	metrics.PublisherAcknowledgementTimeouts: ccsmp.SolClientStatsTXAckTimeout,
	metrics.ControlMessagesSent:              ccsmp.SolClientStatsTXCtlMsgs,
	metrics.ControlBytesSent:                 ccsmp.SolClientStatsTXCtlBytes,
	metrics.ConnectionAttempts:               ccsmp.SolClientStatsTXTotalConnectionAttempts,
	metrics.PublishedMessagesAcknowledged:    ccsmp.SolClientStatsTXGuaranteedMsgsSentConfirmed,
	metrics.PublishMessagesDiscarded:         ccsmp.SolClientStatsTXDiscardChannelError,
	metrics.PublisherWouldBlock:              ccsmp.SolClientStatsTXWouldBlock,
}

var clientMetrics = map[metrics.Metric]NextGenMetric{
	metrics.ReceivedMessagesTerminationDiscarded:  MetricReceivedMessagesTerminationDiscarded,
	metrics.ReceivedMessagesBackpressureDiscarded: MetricReceivedMessagesBackpressureDiscarded,
	metrics.PublishMessagesTerminationDiscarded:   MetricPublishMessagesTerminationDiscarded,
	metrics.PublishMessagesBackpressureDiscarded:  MetricPublishMessagesBackpressureDiscarded,
	metrics.InternalDiscardNotifications:          MetricInternalDiscardNotifications,
}

// NextGenMetric structure
type NextGenMetric int

const (
	// MetricReceivedMessagesTerminationDiscarded initialized
	MetricReceivedMessagesTerminationDiscarded NextGenMetric = iota

	// MetricReceivedMessagesBackpressureDiscarded initialized
	MetricReceivedMessagesBackpressureDiscarded NextGenMetric = iota

	// MetricPublishMessagesTerminationDiscarded initialized
	MetricPublishMessagesTerminationDiscarded NextGenMetric = iota

	// MetricPublishMessagesBackpressureDiscarded initialized
	MetricPublishMessagesBackpressureDiscarded NextGenMetric = iota

	// MetricInternalDiscardNotifications initialized
	MetricInternalDiscardNotifications NextGenMetric = iota

	// metricCount initialized
	metricCount int = iota
)

// Metrics interface
type Metrics interface {
	GetStat(metric metrics.Metric) uint64
	IncrementMetric(metric NextGenMetric, amount uint64)
	ResetStats()
}

// Implementation
type ccsmpBackedMetrics struct {
	session *ccsmp.SolClientSession
	metrics []uint64

	metricLock        sync.RWMutex
	capturedTxMetrics map[ccsmp.SolClientStatsTX]uint64
	capturedRxMetrics map[ccsmp.SolClientStatsRX]uint64
}

func newCcsmpMetrics(session *ccsmp.SolClientSession) *ccsmpBackedMetrics {
	return &ccsmpBackedMetrics{
		metrics: make([]uint64, metricCount),
		session: session,
	}
}

func (metrics *ccsmpBackedMetrics) terminate() {
	metrics.metricLock.Lock()
	defer metrics.metricLock.Unlock()
	metrics.captureTXStats()
	metrics.captureRXStats()
}

func (metrics *ccsmpBackedMetrics) captureTXStats() {
	if metrics.capturedTxMetrics != nil {
		return
	}
	metrics.capturedTxMetrics = make(map[ccsmp.SolClientStatsTX]uint64)
	for _, txStat := range txMetrics {
		metrics.capturedTxMetrics[txStat] = metrics.session.SolClientSessionGetTXStat(txStat)
	}
}

func (metrics *ccsmpBackedMetrics) captureRXStats() {
	if metrics.capturedRxMetrics != nil {
		return
	}
	metrics.capturedRxMetrics = make(map[ccsmp.SolClientStatsRX]uint64)
	for _, rxStat := range rxMetrics {
		metrics.capturedRxMetrics[rxStat] = metrics.session.SolClientSessionGetRXStat(rxStat)
	}
}

func (metrics *ccsmpBackedMetrics) getTXStat(stat ccsmp.SolClientStatsTX) uint64 {
	metrics.metricLock.RLock()
	defer metrics.metricLock.RUnlock()
	if metrics.capturedTxMetrics != nil {
		return metrics.capturedTxMetrics[stat]
	}
	return metrics.session.SolClientSessionGetTXStat(stat)
}

func (metrics *ccsmpBackedMetrics) getRXStat(stat ccsmp.SolClientStatsRX) uint64 {
	metrics.metricLock.RLock()
	defer metrics.metricLock.RUnlock()
	if metrics.capturedRxMetrics != nil {
		return metrics.capturedRxMetrics[stat]
	}
	return metrics.session.SolClientSessionGetRXStat(stat)
}

func (metrics *ccsmpBackedMetrics) getNextGenStat(metric NextGenMetric) uint64 {
	return atomic.LoadUint64(&metrics.metrics[metric])
}

func (metrics *ccsmpBackedMetrics) GetStat(metric metrics.Metric) uint64 {
	if rxMetric, ok := rxMetrics[metric]; ok {
		return metrics.getRXStat(rxMetric)
	} else if txMetric, ok := txMetrics[metric]; ok {
		return metrics.getTXStat(txMetric)
	} else if clientMetric, ok := clientMetrics[metric]; ok {
		return metrics.getNextGenStat(clientMetric)
	}
	logging.Default.Warning("Could not find mapping for metric with ID " + fmt.Sprint(metric))
	return 0

}

func (metrics *ccsmpBackedMetrics) ResetStats() {
	for i := 0; i < metricCount; i++ {
		atomic.StoreUint64(&metrics.metrics[i], 0)
	}
	metrics.resetNativeStats()
}

func (metrics *ccsmpBackedMetrics) resetNativeStats() {
	metrics.metricLock.Lock()
	defer metrics.metricLock.Unlock()
	if metrics.capturedRxMetrics != nil {
		for key := range metrics.capturedRxMetrics {
			metrics.capturedRxMetrics[key] = 0
		}
		for key := range metrics.capturedTxMetrics {
			metrics.capturedTxMetrics[key] = 0
		}
	} else {
		errorInfo := metrics.session.SolClientSessionClearStats()
		if errorInfo != nil {
			logging.Default.Warning("Could not reset metrics: " + errorInfo.String())
		}
	}
}

func (metrics *ccsmpBackedMetrics) IncrementMetric(metric NextGenMetric, amount uint64) {
	atomic.AddUint64(&metrics.metrics[metric], amount)
}
