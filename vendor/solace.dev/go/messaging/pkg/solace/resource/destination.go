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

// Package resource contains types and factory functions for various
// broker resources such as topics and queues.
package resource

import "fmt"

// Destination represents a message destination on a broker.
// Some examples of destinations include queues and topics.
// Destination implementations can be retrieved by the various helper functions,
// such as TopicOf, QueueDurableExclusive, and TopicSubscriptionOf.
type Destination interface {
	// GetName retrieves the name of the destination
	GetName() string
}

// Subscription represents the valid subscriptions that can be specified to receivers.
// Valid subscriptions include *resource.TopicSubscription.
type Subscription interface {
	Destination
	// GetSubscriptionType will return the type of the subscription as a string
	GetSubscriptionType() string
}

// Topic is an implementation of destination representing a topic
// that can be published to.
type Topic struct {
	name string
}

// TopicOf creates a new topic with the specified name.
// Topic name must not be empty.
func TopicOf(expression string) *Topic {
	return &Topic{expression}
}

// GetName returns the name of the topic. Implements the Destination interface.
func (t *Topic) GetName() string {
	return t.name
}

// String implements fmt.Stringer
func (t *Topic) String() string {
	return fmt.Sprintf("Topic: %s", t.GetName())
}

// ShareName is an interface for identifiers that are associated
// with a shared subscription.
// See https://docs.solace.com/PubSub-Basics/Direct-Messages.htm#Shared in the Solace documentation.
type ShareName struct {
	name string
}

// ShareNameOf returns a new share name with the provided name.
// Valid share names are not empty and do not contain special
// characters '>' or '*'. Returns a new ShareName with the given string.
func ShareNameOf(name string) *ShareName {
	return &ShareName{name}
}

// GetName returns the share name. Implements the Destination interface.
func (sn *ShareName) GetName() string {
	return sn.name
}

// String implements fmt.Stringer
func (sn *ShareName) String() string {
	return fmt.Sprintf("ShareName: %s", sn.GetName())
}

// TopicSubscription is a subscription to a topic often used for receivers.
type TopicSubscription struct {
	topic string
}

// GetName returns the topic subscription expression. Implements the Destination interface.
func (t *TopicSubscription) GetName() string {
	return t.topic
}

// GetSubscriptionType returns the type of the topic subscription as a string
func (t *TopicSubscription) GetSubscriptionType() string {
	return fmt.Sprintf("%T", t)
}

// String implements fmt.Stringer
func (t *TopicSubscription) String() string {
	return fmt.Sprintf("TopicSubscription: %s", t.GetName())
}

// TopicSubscriptionOf creates a TopicSubscription of the specified topic string.
func TopicSubscriptionOf(topic string) *TopicSubscription {
	return &TopicSubscription{topic}
}

// Queue represents a queue used for guaranteed messaging receivers.
type Queue struct {
	name                           string
	exclusivelyAccessible, durable bool
}

// GetName returns the name of the queue. Implements the Destination interface.
func (q *Queue) GetName() string {
	return q.name
}

// IsExclusivelyAccessible determines if Queue supports exclusive or shared-access mode.
// Returns true if the Queue can serve only one consumer at any one time, false if the
// Queue can serve multiple consumers with each consumer serviced in a round-robin fashion.
func (q *Queue) IsExclusivelyAccessible() bool {
	return q.exclusivelyAccessible
}

// IsDurable determines if the Queue is durable. Durable queues are privisioned objects on
// the broker that have a lifespan that is independent of any one client session.
func (q *Queue) IsDurable() bool {
	return q.durable
}

func (q *Queue) String() string {
	return fmt.Sprintf("Queue: %s, exclusive: %t, durable: %t", q.GetName(), q.IsExclusivelyAccessible(), q.IsDurable())
}

// QueueDurableExclusive creates a new durable, exclusive queue with the specified name.
func QueueDurableExclusive(queueName string) *Queue {
	return &Queue{
		name:                  queueName,
		exclusivelyAccessible: true,
		durable:               true,
	}
}

// QueueDurableNonExclusive creates a durable, non-exclusive queue with the specified name.
func QueueDurableNonExclusive(queueName string) *Queue {
	return &Queue{
		name:                  queueName,
		exclusivelyAccessible: false,
		durable:               true,
	}
}

// QueueNonDurableExclusive creates an exclusive, non-durable queue with the specified name.
func QueueNonDurableExclusive(queueName string) *Queue {
	return &Queue{
		name:                  queueName,
		exclusivelyAccessible: true,
		durable:               false,
	}
}

// QueueNonDurableExclusiveAnonymous creates an anonymous, exclusive, and non-durable queue.
func QueueNonDurableExclusiveAnonymous() *Queue {
	return &Queue{
		name:                  "",
		exclusivelyAccessible: true,
		durable:               false,
	}
}
