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

package constants

// Constants used for various error messages should go here
// Note that errors in golang by convention start with a lowercase letter and do not end in punctuation

// UnableToConnectAlreadyDisconnectedService error string
const UnableToConnectAlreadyDisconnectedService = "unable to connect messaging service in state %s"

// UnableToConnectAlreadyDisposedService error string
const UnableToConnectAlreadyDisposedService = "unable to connect messaging service after it has already been disposed of"

// UnableToDisconnectUnstartedService error string
const UnableToDisconnectUnstartedService = "unable to disconnect messaging service in state %s"

// UnableToPublishAlreadyTerminated error string
const UnableToPublishAlreadyTerminated = "unable to publish message: message publisher has been terminated"

// UnableToPublishNotStarted error string
const UnableToPublishNotStarted = "unable to publish message: message publisher is not started. current state: "

// UnableToTerminatePublisher error string
const UnableToTerminatePublisher = "cannot terminate the publisher as it has not been started"

// UnableToStartPublisher error string
const UnableToStartPublisher = "cannot start the publisher as it has already been terminated"

// UnableToStartPublisherParentServiceNotStarted error string
const UnableToStartPublisherParentServiceNotStarted = "cannot start publisher unless parent MessagingService is connected"

// UnableToTerminateReceiver error string
const UnableToTerminateReceiver = "cannot terminate the receiver as it has not been started"

// UnableToStartReceiver error string
const UnableToStartReceiver = "cannot start the receiver as it has already been terminated"

// UnableToStartReceiverParentServiceNotStarted error string
const UnableToStartReceiverParentServiceNotStarted = "cannot start receiver unless parent MessagingService is connected"

// UnableToAcknowledgeAlreadyTerminated error string
const UnableToAcknowledgeAlreadyTerminated = "unable to acknowledge message: message receiver has been terminated"

// UnableToAcknowledgeNotStarted error string
const UnableToAcknowledgeNotStarted = "unable to acknowledge meessage: message receiver is not yet started"

// UnableToModifySubscriptionBadState error string
const UnableToModifySubscriptionBadState = "unable to modify subscriptions in state %s"

// DirectReceiverUnsupportedSubscriptionType error string
const DirectReceiverUnsupportedSubscriptionType = "DirectMessageReceiver does not support subscriptions of type %T"

// ReceiverTimedOutWaitingForMessage error string
const ReceiverTimedOutWaitingForMessage = "timed out waiting for message on call to Receive"

// ReceiverCannotReceiveNotStarted error string
const ReceiverCannotReceiveNotStarted = "receiver has not yet been started, no messages to receive"

// ReceiverCannotReceiveAlreadyTerminated error string
const ReceiverCannotReceiveAlreadyTerminated = "receiver has been terminated, no messages to receive"

// PersistentReceiverUnsupportedSubscriptionType error string
const PersistentReceiverUnsupportedSubscriptionType = "PersistentMessageReceiver does not support subscriptions of type %T"

// PersistentReceiverMissingQueue error string
const PersistentReceiverMissingQueue = "queue must be provided when building a new PersistentMessageReceiver"

// PersistentReceiverCannotPauseBadState error string
const PersistentReceiverCannotPauseBadState = "cannot pause message receiption when not in started state"

// PersistentReceiverCannotUnpauseBadState error string
const PersistentReceiverCannotUnpauseBadState = "cannot resume message receiption when not in started state"

// PersistentReceiverMustSpecifyRGMID error string
const PersistentReceiverMustSpecifyRGMID = "must specify ReceiverPropertyPersistentMessageReplayStrategyIDBasedReplicationGroupMessageID when replay from message ID is selected"

// PersistentReceiverMustSpecifyTime error string
const PersistentReceiverMustSpecifyTime = "must specify ReceiverPropertyPersistentMessageReplayStrategyTimeBasedStartTime when replay from time is selected"

// WouldBlock error string
const WouldBlock = "buffer is full, cannot queue additional messages"

// UnableToRetrieveMessageID error string
const UnableToRetrieveMessageID = "unable to retrieve message ID from message, was the message received by a persistent receiver?"

// IncompleteMessageDeliveryMessage error string
const IncompleteMessageDeliveryMessage = "failed to publish messages, publisher terminated with %d undelivered messages"

// IncompleteMessageDeliveryMessageWithUnacked error string
const IncompleteMessageDeliveryMessageWithUnacked = "failed to publish messages, publisher terminated with %d undelivered messages and %d unacknowledged messages"

// IncompleteMessageReceptionMessage error string
const IncompleteMessageReceptionMessage = "failed to dispatch messages, receiver terminated with %d undelivered messages"

// LoginFailure error string
const LoginFailure = "failed to authenticate with the broker: "

// UnresolvedSession error string
const UnresolvedSession = "requested service is unreachable: "

// DirectReceiverBackpressureMustBeGreaterThan0 error string
const DirectReceiverBackpressureMustBeGreaterThan0 = "direct receiver backpressure buffer size must be > 0"

// UnableToRegisterCallbackReceiverTerminating error string
const UnableToRegisterCallbackReceiverTerminating = "cannot register message handler, receiver is not running"

// FailedToAddSubscription error string
const FailedToAddSubscription = "failed to add subscription to receiver: "

// InvalidUserPropertyDataType error string
const InvalidUserPropertyDataType = "type %T at key %s is not supported for user data"

// InvalidOutboundMessageType error string
const InvalidOutboundMessageType = "got an invalid OutboundMessage, was it built by OutboundMessageBuilder? backing type %T"

// InvalidInboundMessageType error string
const InvalidInboundMessageType = "got an invalid InboundMessage, was it received with a receiver? backing type %T"

// TerminatedOnMessagingServiceShutdown error string
const TerminatedOnMessagingServiceShutdown = "terminated due to MessagingService disconnect"

// ShareNameMustNotBeEmpty error string
const ShareNameMustNotBeEmpty = "share name must not be empty"

// ShareNameMustNotContainInvalidCharacters error string
const ShareNameMustNotContainInvalidCharacters = "share name must not contain literals '*' or '>'"

// MissingServiceProperty error string
const MissingServiceProperty = "property %s is required to build a messaging service"

// MissingServicePropertyGivenScheme error string
const MissingServicePropertyGivenScheme = "property %s is required when using authentication scheme %s"

// MissingServicePropertiesGivenScheme error string
const MissingServicePropertiesGivenScheme = "one of properties %s is required when using authentication scheme %s"

// CouldNotCreateRGMID error string
const CouldNotCreateRGMID = "could not create ReplicationGroupMessageID from string: %s"

// CouldNotCompareRGMID error string
const CouldNotCompareRGMID = "could not compare ReplicationGroupMessageIDs: %s"

// CouldNotCompareRGMIDBadType error string
const CouldNotCompareRGMIDBadType = "could not compare with ReplicationGroupMessageID of type %T"

// CouldNotConfirmSubscriptionServiceUnavailable error string
const CouldNotConfirmSubscriptionServiceUnavailable = "could not confirm subscription, the messaging service was terminated"

// InvalidConfiguration error string
const InvalidConfiguration = "invalid configuration provided: "
