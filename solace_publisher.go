package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"solace.dev/go/messaging"
	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/resource"
)

func main() {
	// Configuration parameters
	brokerConfig := config.ServicePropertyMap{
		config.TransportLayerPropertyHost:                "tcp://public.messaging.solace.cloud",
		config.ServicePropertyVPNName:                    "public",
		config.AuthenticationPropertySchemeBasicUserName: "conf42",
		config.AuthenticationPropertySchemeBasicPassword: "public",
	}

	messagingService, err := messaging.NewMessagingServiceBuilder().FromConfigurationProvider(brokerConfig).Build()

	if err != nil {
		panic(err)
	}

	// Connect to the messaging serice
	if err := messagingService.Connect(); err != nil {
		panic(err)
	}

	//  Build a Direct Message Publisher
	directPublisher, builderErr := messagingService.CreateDirectMessagePublisherBuilder().Build()
	if builderErr != nil {
		panic(builderErr)
	}

	// Start the publisher
	startErr := directPublisher.Start()
	if startErr != nil {
		panic(startErr)
	}

	msgSeqNum := 0

	//  Prepare outbound message payload and body
	messageBody := "Hello from Conf42"
	messageBuilder := messagingService.MessageBuilder().
		WithProperty("application", "samples").
		WithProperty("language", "go")

	// Run forever until an interrupt signal is received
	go func() {
		for directPublisher.IsReady() {
			msgSeqNum++
			message, err := messageBuilder.BuildWithStringPayload(messageBody + " --> " + strconv.Itoa(msgSeqNum))
			if err != nil {
				panic(err)
			}
			topic := resource.TopicOf("conf42/solace/go/" + strconv.Itoa(msgSeqNum))

			// Publish on dynamic topic with dynamic body
			publishErr := directPublisher.Publish(message, topic)
			if publishErr != nil {
				panic(publishErr)
			}

			fmt.Println("Published message on topic: ", topic.GetName())
			time.Sleep(1 * time.Second)
		}
	}()

	// Handle OS interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until an OS interrupt signal is received.
	<-c

	// Terminate the Direct Receiver
	directPublisher.Terminate(1 * time.Second)
	fmt.Println("\nDirect Publisher Terminated? ", directPublisher.IsTerminated())
	// Disconnect the Message Service
	messagingService.Disconnect()
	fmt.Println("Messaging Service Disconnected? ", !messagingService.IsConnected())

}
