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

package subcode

// Code generated by subcode_generator.go via go generate. DO NOT EDIT.
const (
	// Ok: No error.
	Ok Code = 0
	// ParamOutOfRange: An API call was made with an out-of-range parameter.
	ParamOutOfRange Code = 1
	// ParamNullPtr: An API call was made with a null or invalid pointer parameter.
	ParamNullPtr Code = 2
	// ParamConflict: An API call was made with a parameter combination that is not valid.
	ParamConflict Code = 3
	// InsufficientSpace: An API call failed due to insufficient space to accept more data.
	InsufficientSpace Code = 4
	// OutOfResources: An API call failed due to lack of resources (for example, starting a timer when all timers are in use).
	OutOfResources Code = 5
	// InternalError: An API call had an internal error (not an application fault).
	InternalError Code = 6
	// OutOfMemory: An API call failed due to inability to allocate memory.
	OutOfMemory Code = 7
	// ProtocolError: An API call failed due to a protocol error with the appliance (not an application fault).
	ProtocolError Code = 8
	// InitNotCalled: An API call failed due to solClient_initialize() not being called first.
	InitNotCalled Code = 9
	// Timeout: An API call failed due to a timeout.
	Timeout Code = 10
	// KeepAliveFailure: The Session Keep-Alive detected a failed Session.
	KeepAliveFailure Code = 11
	// SessionNotEstablished: An API call failed due to the Session not being established.
	SessionNotEstablished Code = 12
	// OsError: An API call failed due to a failed operating system call; an error string can be retrieved with solClient_getLastErrorInfo().
	OsError Code = 13
	// CommunicationError: An API call failed due to a communication error. An error string can be retrieved with solClient_getLastErrorInfo().
	CommunicationError Code = 14
	// UserDataTooLarge: An attempt was made to send a message with user data larger than the maximum that is supported.
	UserDataTooLarge Code = 15
	// TopicTooLarge: An attempt was made to use a Topic that is longer than the maximum that is supported.
	TopicTooLarge Code = 16
	// InvalidTopicSyntax: An attempt was made to use a Topic that has a syntax which is not supported.
	InvalidTopicSyntax Code = 17
	// XmlParseError: The appliance could not parse an XML message.
	XmlParseError Code = 18
	// LoginFailure: The client could not log into the appliance (bad username or password).
	LoginFailure Code = 19
	// InvalidVirtualAddress: An attempt was made to connect to the wrong IP address on the appliance (must use CVRID if configured) or the appliance CVRID has changed and this was detected on reconnect.
	InvalidVirtualAddress Code = 20
	// ClientDeleteInProgress: The client login not currently possible as previous instance of same client still being deleted.
	ClientDeleteInProgress Code = 21
	// TooManyClients: The client login not currently possible becuase the maximum number of active clients on appliance has already been reached.
	TooManyClients Code = 22
	// SubscriptionAlreadyPresent: The client attempted to add a subscription which already exists. This subcode is only returned if the Session property SOLCLIENT_SESSION_PROP_IGNORE_DUP_SUBSCRIPTION_ERROR is not enabled.
	SubscriptionAlreadyPresent Code = 23
	// SubscriptionNotFound: The client attempted to remove a subscription which did not exist. This subcode is only returned if the Session property SOLCLIENT_SESSION_PROP_IGNORE_DUP_SUBSCRIPTION_ERROR is not enabled.
	SubscriptionNotFound Code = 24
	// SubscriptionInvalid: The client attempted to add/remove a subscription that is not valid.
	SubscriptionInvalid Code = 25
	// SubscriptionOther: The appliance rejected a subscription add or remove request for a reason not separately enumerated.
	SubscriptionOther Code = 26
	// ControlOther: The appliance rejected a control message for another reason not separately enumerated.
	ControlOther Code = 27
	// DataOther: The appliance rejected a data message for another reason not separately enumerated.
	DataOther Code = 28
	// LogFileError: Could not open the log file name specified by the application for writing (Deprecated - ::SOLCLIENT_SUBCODE_OS_ERROR is used).
	LogFileError Code = 29
	// MessageTooLarge: The client attempted to send a message larger than that supported by the appliance.
	MessageTooLarge Code = 30
	// SubscriptionTooMany: The client attempted to add a subscription that exceeded the maximum number allowed.
	SubscriptionTooMany Code = 31
	// InvalidSessionOperation: An API call failed due to the attempted operation not being valid for the Session.
	InvalidSessionOperation Code = 32
	// TopicMissing: A send call was made that did not have a Topic in a mode where one is required (for example, client mode).
	TopicMissing Code = 33
	// AssuredMessagingNotEstablished: A send call was made to send a Guaranteed message before Guaranteed Delivery is established (Deprecated).
	AssuredMessagingNotEstablished Code = 34
	// AssuredMessagingStateError: An attempt was made to start Guaranteed Delivery when it is already started.
	AssuredMessagingStateError Code = 35
	// QueuenameTopicConflict: Both Queue Name and Topic are specified in solClient_session_send.
	QueuenameTopicConflict Code = 36
	// QueuenameTooLarge: An attempt was made to use a Queue name which is longer than the maximum supported length.
	QueuenameTooLarge Code = 37
	// QueuenameInvalidMode: An attempt was made to use a Queue name on a non-Guaranteed message.
	QueuenameInvalidMode Code = 38
	// MaxTotalMsgsizeExceeded: An attempt was made to send a message with a total size greater than that supported by the protocol.
	MaxTotalMsgsizeExceeded Code = 39
	// DblockAlreadyExists: An attempt was made to allocate a datablock for a msg element when one already exists.
	DblockAlreadyExists Code = 40
	// NoStructuredData: An attempt was made to create a container to read structured data where none exists.
	NoStructuredData Code = 41
	// ContainerBusy: An attempt was made to add a field to a map or stream while a sub map or stream is being built.
	ContainerBusy Code = 42
	// InvalidDataConversion: An attempt was made to retrieve structured data with wrong type.
	InvalidDataConversion Code = 43
	// CannotModifyWhileNotIdle: An attempt was made to modify a property that cannot be modified while Session is not idle.
	CannotModifyWhileNotIdle Code = 44
	// MsgVpnNotAllowed: The Message VPN name configured for the session does not exist.
	MsgVpnNotAllowed Code = 45
	// ClientNameInvalid: The client name chosen has been rejected as invalid by the appliance.
	ClientNameInvalid Code = 46
	// MsgVpnUnavailable: The Message VPN name set for the Session (or the default Message VPN, if none was set) is currently shutdown on the appliance.
	MsgVpnUnavailable Code = 47
	// ClientUsernameIsShutdown: The username for the client is administratively shutdown on the appliance.
	ClientUsernameIsShutdown Code = 48
	// DynamicClientsNotAllowed: The username for the Session has not been set and dynamic clients are not allowed.
	DynamicClientsNotAllowed Code = 49
	// ClientNameAlreadyInUse: The Session is attempting to use a client, publisher name, or subscriber name that is in use by another client, publisher, or subscriber, and the appliance is configured to reject the new Session. When Message VPNs are in use, the conflicting client name must be in the same Message VPN.
	ClientNameAlreadyInUse Code = 50
	// CacheNoData: When the cache request returns ::SOLCLIENT_INCOMPLETE, this subcode indicates there is no cached data in the designated cache.
	CacheNoData Code = 51
	// CacheSuspectData: When the designated cache responds to a cache request with suspect data the API returns ::SOLCLIENT_INCOMPLETE with this subcode.
	CacheSuspectData Code = 52
	// CacheErrorResponse: The cache instance has returned an error response to the request.
	CacheErrorResponse Code = 53
	// CacheInvalidSession: The cache session operation failed because the Session has been destroyed.
	CacheInvalidSession Code = 54
	// CacheTimeout: The cache session operation failed because the request timeout expired.
	CacheTimeout Code = 55
	// CacheLivedataFulfill: The cache session operation completed when live data arrived on the Topic requested.
	CacheLivedataFulfill Code = 56
	// CacheAlreadyInProgress: A cache request has been made when there is already a cache request outstanding on the same Topic and SOLCLIENT_CACHEREQUEST_FLAGS_LIVEDATA_FLOWTHRU was not set.
	CacheAlreadyInProgress Code = 57
	// MissingReplyTo: A message does not have the required reply-to field.
	MissingReplyTo Code = 58
	// CannotBindToQueue: Already bound to the queue, or not authorized to bind to the queue.
	CannotBindToQueue Code = 59
	// InvalidTopicNameForTe: An attempt was made to bind to a Topic Endpoint with an invalid topic.
	InvalidTopicNameForTe Code = 60
	// UnknownQueueName: An attempt was made to bind to an unknown Queue name (for example, not configured on appliance).
	UnknownQueueName Code = 61
	// UnknownTeName: An attempt was made to bind to an unknown Topic Endpoint name (for example, not configured on appliance).
	UnknownTeName Code = 62
	// MaxClientsForQueue: An attempt was made to bind to a Queue that already has a maximum number of clients.
	MaxClientsForQueue Code = 63
	// MaxClientsForTe: An attempt was made to bind to a Topic Endpoint that already has a maximum number of clients.
	MaxClientsForTe Code = 64
	// UnexpectedUnbind: An unexpected unbind response was received for a Queue or Topic Endpoint (for example, the Queue or Topic Endpoint was deleted from the appliance).
	UnexpectedUnbind Code = 65
	// QueueNotFound: The specified Queue was not found when publishing a message.
	QueueNotFound Code = 66
	// ClientAclDenied: The client login to the appliance was denied because the IP address/netmask combination used for the client is designated in the ACL (Access Control List) as a deny connection for the given Message VPN and username.
	ClientAclDenied Code = 67
	// SubscriptionAclDenied: Adding a subscription was denied because it matched a subscription that was defined on the ACL (Access Control List).
	SubscriptionAclDenied Code = 68
	// PublishAclDenied: A message could not be published because its Topic matched a Topic defined on the ACL (Access Control List).
	PublishAclDenied Code = 69
	// DeliverToOneInvalid: An attempt was made to set both Deliver-To-One (DTO) and Guaranteed Delivery in the same message. (Deprecated:  DTO will be applied to the corresponding demoted direct message)
	DeliverToOneInvalid Code = 70
	// SpoolOverQuota: Message was not delivered because the Guaranteed message spool is over its allotted space quota.
	SpoolOverQuota Code = 71
	// QueueShutdown: An attempt was made to operate on a shutdown queue.
	QueueShutdown Code = 72
	// TeShutdown: An attempt was made to bind to a shutdown Topic Endpoint.
	TeShutdown Code = 73
	// NoMoreNonDurableQueueOrTe: An attempt was made to bind to a non-durable Queue or Topic Endpoint, and the appliance is out of resources.
	NoMoreNonDurableQueueOrTe Code = 74
	// EndpointAlreadyExists: An attempt was made to create a Queue or Topic Endpoint that already exists. This subcode is only returned if the provision flag SOLCLIENT_PROVISION_FLAGS_IGNORE_EXIST_ERRORS is not set.
	EndpointAlreadyExists Code = 75
	// PermissionNotAllowed: An attempt was made to delete or create a Queue or Topic Endpoint when the Session does not have authorization for the action. This subcode is also returned when an attempt is made to remove a message from an endpoint when the Session does not have 'consume' authorization, or when an attempt is made to add or remove a Topic subscription from a Queue when the Session does not have 'modify-topic' authorization.
	PermissionNotAllowed Code = 76
	// InvalidSelector: An attempt was made to bind to a Queue or Topic Endpoint with an invalid selector.
	InvalidSelector Code = 77
	// MaxMessageUsageExceeded: Publishing of message denied because the maximum spooled message count was exceeded.
	MaxMessageUsageExceeded Code = 78
	// EndpointPropertyMismatch: An attempt was made to create a dynamic durable endpoint and it was found to exist with different properties.
	EndpointPropertyMismatch Code = 79
	// SubscriptionManagerDenied: An attempt was made to add a subscription to another client when Session does not have subscription manager privileges.
	SubscriptionManagerDenied Code = 80
	// UnknownClientName: An attempt was made to add a subscription to another client that is unknown on the appliance.
	UnknownClientName Code = 81
	// QuotaOutOfRange: An attempt was made to provision an endpoint with a quota that is out of range.
	QuotaOutOfRange Code = 82
	// SubscriptionAttributesConflict: The client attempted to add a subscription which already exists but it has different properties
	SubscriptionAttributesConflict Code = 83
	// InvalidSmfMessage: The client attempted to send a Solace Message Format (SMF) message using solClient_session_sendSmf() or solClient_session_sendMultipleSmf(), but the buffer did not contain a Direct message.
	InvalidSmfMessage Code = 84
	// NoLocalNotSupported: The client attempted to establish a Session or Flow with No Local enabled and the capability is not supported by the appliance.
	NoLocalNotSupported Code = 85
	// UnsubscribeNotAllowedClientsBound: The client attempted to unsubscribe a Topic from a Topic Endpoint while there were still Flows bound to the endpoint.
	UnsubscribeNotAllowedClientsBound Code = 86
	// CannotBlockInContext: An API function was invoked in the Context thread that would have blocked otherwise. For an example, a call may have been made to send a message when the Session is configured with ::SOLCLIENT_SESSION_PROP_SEND_BLOCKING enabled and the transport (socket or IPC) channel is full. All application callback functions are executed in the Context thread.
	CannotBlockInContext Code = 87
	// FlowActiveFlowIndicationUnsupported: The client attempted to establish a Flow with Active Flow Indication (SOLCLIENT_FLOW_PROP_ACTIVE_FLOW_IND) enabled and the capability is not supported by the appliance
	FlowActiveFlowIndicationUnsupported Code = 88
	// UnresolvedHost: The client failed to connect because the host name could not be resolved.
	UnresolvedHost Code = 89
	// CutThroughUnsupported: An attempt was made to create a 'cut-through' Flow on a Session that does not support this capability
	CutThroughUnsupported Code = 90
	// CutThroughAlreadyBound: An attempt was made to create a 'cut-through' Flow on a Session that already has one 'cut-through' Flow
	CutThroughAlreadyBound Code = 91
	// CutThroughIncompatibleWithSession: An attempt was made to create a 'cut-through' Flow on a Session with incompatible Session properties. Cut-through may not be enabled on Sessions with SOLCLIENT_SESSION_PROP_TOPIC_DISPATCH enabled.
	CutThroughIncompatibleWithSession Code = 92
	// InvalidFlowOperation: An API call failed due to the attempted operation not being valid for the Flow.
	InvalidFlowOperation Code = 93
	// UnknownFlowName: The session was disconnected due to loss of the publisher flow state. All (unacked and unsent) messages held by the API were deleted. To connect the session, applications need to call ::solClient_session_connect again.
	UnknownFlowName Code = 94
	// ReplicationIsStandby: An attempt to perform an operation using a VPN that is configured to be STANDBY for replication.
	ReplicationIsStandby Code = 95
	// LowPriorityMsgCongestion: The message was rejected by the appliance as one or more matching endpoints exceeded the reject-low-priority-msg-limit.
	LowPriorityMsgCongestion Code = 96
	// LibraryNotLoaded: The client failed to find the library or symbol.
	LibraryNotLoaded Code = 97
	// FailedLoadingTruststore: The client failed to load the trust store.
	FailedLoadingTruststore Code = 98
	// UntrustedCertificate: The client attempted to connect to an appliance that has a suspect certficate.
	UntrustedCertificate Code = 99
	// UntrustedCommonname: The client attempted to connect to an appliance that has a suspect common name.
	UntrustedCommonname Code = 100
	// CertificateDateInvalid: The client attempted to connect to an appliance that does not have a valid certificate date.
	CertificateDateInvalid Code = 101
	// FailedLoadingCertificateAndKey: The client failed to load certificate and/or private key files.
	FailedLoadingCertificateAndKey Code = 102
	// BasicAuthenticationIsShutdown:  The client attempted to connect to an appliance that has the basic authentication shutdown.
	BasicAuthenticationIsShutdown Code = 103
	// ClientCertificateAuthenticationIsShutdown:  The client attempted to connect to an appliance that has the client certificate authentication shutdown.
	ClientCertificateAuthenticationIsShutdown Code = 104
	// UntrustedClientCertificate: The client failed to connect to an appliance as it has a suspect client certificate.
	UntrustedClientCertificate Code = 105
	// ClientCertificateDateInvalid: The client failed to connect to an appliance as it does not have a valid client certificate date.
	ClientCertificateDateInvalid Code = 106
	// CacheRequestCancelled: The cache request has been cancelled by the client.
	CacheRequestCancelled Code = 107
	// DeliveryModeUnsupported: Attempt was made from a Transacted Session to send a message with the delivery mode SOLCLIENT_DELIVERY_MODE_DIRECT.
	DeliveryModeUnsupported Code = 108
	// PublisherNotCreated: Client attempted to send a message from a Transacted Session without creating a default publisher flow.
	PublisherNotCreated Code = 109
	// FlowUnbound: The client attempted to receive message from an UNBOUND Flow with no queued messages in memory.
	FlowUnbound Code = 110
	// InvalidTransactedSessionID: The client attempted to commit or rollback a transaction with an invalid Transacted Session Id.
	InvalidTransactedSessionID Code = 111
	// InvalidTransactionID: The client attempted to commit or rollback a transaction with an invalid transaction Id.
	InvalidTransactionID Code = 112
	// MaxTransactedSessionsExceeded: The client failed to open a Transacted Session as it exceeded the max Transacted Sessions.
	MaxTransactedSessionsExceeded Code = 113
	// TransactedSessionNameInUse: The client failed to open a Transacted Session as the Transacted Session name provided is being used by another opened session.
	TransactedSessionNameInUse Code = 114
	// ServiceUnavailable: Guaranteed Delivery services are not enabled on the appliance.
	ServiceUnavailable Code = 115
	// NoTransactionStarted: The client attempted to commit an unknown transaction.
	NoTransactionStarted Code = 116
	// PublisherNotEstablished: A send call was made on a transacted session before its publisher is established.
	PublisherNotEstablished Code = 117
	// MessagePublishFailure: The client attempted to commit a transaction with a GD publish failure encountered.
	MessagePublishFailure Code = 118
	// TransactionFailure: The client attempted to commit a transaction with too many transaction steps.
	TransactionFailure Code = 119
	// MessageConsumeFailure: The client attempted to commit a transaction with a consume failure encountered.
	MessageConsumeFailure Code = 120
	// EndpointModified: The client attempted to commit a transaction with an Endpoint being shutdown or deleted.
	EndpointModified Code = 121
	// InvalidConnectionOwner: The client attempted to commit a transaction with an unknown connection ID.
	InvalidConnectionOwner Code = 122
	// KerberosAuthenticationIsShutdown: The client attempted to connect to an appliance that has the Kerberos authentication shutdown.
	KerberosAuthenticationIsShutdown Code = 123
	// CommitOrRollbackInProgress: The client attempted to send/receive a message or commit/rollback a transaction when a transaction commit/rollback is in progress.
	CommitOrRollbackInProgress Code = 124
	// UnbindResponseLost: The application called solClient_flow_destroy() and the unbind-response was not received.
	UnbindResponseLost Code = 125
	// MaxTransactionsExceeded: The client failed to open a Transacted Session as the maximum number of transactions was exceeded.
	MaxTransactionsExceeded Code = 126
	// CommitStatusUnknown: The commit response was lost due to a transport layer reconnection to an alternate host in the host list.
	CommitStatusUnknown Code = 127
	// ProxyAuthRequired: The host entry did not contain proxy authentication when required by the proxy server.
	ProxyAuthRequired Code = 128
	// ProxyAuthFailure: The host entry contained invalid proxy authentication when required by the proxy server.
	ProxyAuthFailure Code = 129
	// NoSubscriptionMatch: The client attempted to publish a guaranteed message to a topic that did not have any guaranteed subscription matches or only matched a replicated topic.
	NoSubscriptionMatch Code = 130
	// SubscriptionMatchError: The client attempted to bind to a non-exclusive topic endpoint that is already bound with a different subscription.
	SubscriptionMatchError Code = 131
	// SelectorMatchError: The client attempted to bind to a non-exclusive topic endpoint that is already bound with    a different ingress selector.
	SelectorMatchError Code = 132
	// ReplayNotSupported: Replay is not supported on the Solace Message Router.
	ReplayNotSupported Code = 133
	// ReplayDisabled: Replay is not enabled in the message-vpn.
	ReplayDisabled Code = 134
	// ClientInitiatedReplayNonExclusiveNotAllowed: The client attempted to start replay on a flow bound to a non-exclusive endpoint.
	ClientInitiatedReplayNonExclusiveNotAllowed Code = 135
	// ClientInitiatedReplayInactiveFlowNotAllowed: The client attempted to start replay on an inactive flow.
	ClientInitiatedReplayInactiveFlowNotAllowed Code = 136
	// ClientInitiatedReplayBrowserFlowNotAllowed: The client attempted to bind with both ::SOLCLIENT_FLOW_PROP_BROWSER enabled and ::SOLCLIENT_FLOW_PROP_REPLAY_START_LOCATION set.
	ClientInitiatedReplayBrowserFlowNotAllowed Code = 137
	// ReplayTemporaryNotSupported: Replay is not supported on temporary endpoints.
	ReplayTemporaryNotSupported Code = 138
	// UnknownStartLocationType: The client attempted to start a replay but provided an unknown start location type.
	UnknownStartLocationType Code = 139
	// ReplayMessageUnavailable: A replay in progress on a flow failed because messages to be replayed were trimmed from the replay log.
	ReplayMessageUnavailable Code = 140
	// ReplayStarted: A replay was started on the queue/topic endpoint, either by another client or by an adminstrator on the message router.
	ReplayStarted Code = 141
	// ReplayCancelled: A replay in progress on a flow was administratively cancelled, causing the flow to be unbound.
	ReplayCancelled Code = 142
	// ReplayStartTimeNotAvailable: A replay was requested but the requested start time is not available in the replay log.
	ReplayStartTimeNotAvailable Code = 143
	// ReplayMessageRejected: The Solace Message Router attempted to replay a message, but the queue/topic endpoint rejected the message to the sender.
	ReplayMessageRejected Code = 144
	// ReplayLogModified: A replay in progress on a flow failed because the replay log was modified.
	ReplayLogModified Code = 145
	// MismatchedEndpointErrorID: Endpoint error ID in the bind request does not match the endpoint's error ID.
	MismatchedEndpointErrorID Code = 146
	// OutOfReplayResources: A replay was requested, but the router does not have sufficient resources to fulfill the request, due to too many active replays.
	OutOfReplayResources Code = 147
	// TopicOrSelectorModifiedOnDurableTopicEndpoint: A replay was in progress on a Durable Topic Endpoint (DTE) when its topic or selector was modified, causing the replay to fail.
	TopicOrSelectorModifiedOnDurableTopicEndpoint Code = 148
	// ReplayFailed: A replay in progress on a flow failed.
	ReplayFailed Code = 149
	// CompressedSslNotSupported: The client attempted to establish a Session or Flow with ssl and compression, but the capability is not supported by the appliance.
	CompressedSslNotSupported Code = 150
	// SharedSubscriptionsNotSupported: The client attempted to add a shared subscription, but the capability is not supported by the appliance.
	SharedSubscriptionsNotSupported Code = 151
	// SharedSubscriptionsNotAllowed: The client attempted to add a shared subscription on a client that is not permitted to use shared subscriptions.
	SharedSubscriptionsNotAllowed Code = 152
	// SharedSubscriptionsEndpointNotAllowed: The client attempted to add a shared subscription to a queue or topic endpoint.
	SharedSubscriptionsEndpointNotAllowed Code = 153
	// ObjectDestroyed: The operation cannot be completed because the object (context, session, flow) for the method has been destroyed in another thread.
	ObjectDestroyed Code = 154
	// DeliveryCountNotSupported: The message was received from endpoint that does not support delivery count
	DeliveryCountNotSupported Code = 155
	// ReplayStartMessageUnavailable: A replay was requested but the requested start message is not available in the replay log.
	ReplayStartMessageUnavailable Code = 156
	// MessageIDNotComparable: Replication Group Message Id are not comparable. Messages must be published to the same broker or HA pair for their Replicaton Group Message Id to be comparable.
	MessageIDNotComparable Code = 157
)