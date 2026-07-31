package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/mdp/qrterminal/v3"
	_ "modernc.org/sqlite"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var waClient *whatsmeow.Client

func connectionWhatsApp() (*whatsmeow.Client, error) {

	logger := waLog.Stdout("WhatsApp", "INFO", true)

	ctx := context.Background()

	container, err := sqlstore.New(
		ctx,
		"sqlite",
		"file:whatsapp.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)",
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("create sql store: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("load device: %w", err)
	}

	client := whatsmeow.NewClient(device, logger)

	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {

		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			return nil, err
		}

		if err := client.Connect(); err != nil {
			return nil, err
		}

		for evt := range qrChan {

			switch evt.Event {

			case "code":
				fmt.Println()
				fmt.Println("Scan this QR code with WhatsApp (NEW):")
				fmt.Println()

				qrterminal.GenerateHalfBlock(
					evt.Code,
					qrterminal.L,
					os.Stdout,
				)

			case "success":
				log.Println("WhatsApp login successful.")

			case "timeout":
				return nil, fmt.Errorf("QR code expired")

			case "error":
				return nil, fmt.Errorf("login failed")

			}
		}

	} else {

		// Existing session.
		if err := client.Connect(); err != nil {
			return nil, err
		}

		log.Println("Connected using stored session.")
	}

	waClient = client

	return client, nil
}

func disconnect(client *whatsmeow.Client) {

	if client == nil {
		return
	}

	client.Disconnect()
}

func waitForShutdown(client *whatsmeow.Client) {

	c := make(chan os.Signal, 1)

	signal.Notify(
		c,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-c

	log.Println("Disconnecting WhatsApp...")

	client.Disconnect()
}

func eventHandler(evt interface{}) {

	switch v := evt.(type) {

	case *events.Connected:
		log.Println("WhatsApp connected.")

	case *events.Disconnected:
		log.Println("WhatsApp disconnected.")

	case *events.LoggedOut:
		log.Println("Logged out:", v.Reason)

	case *events.StreamReplaced:
		log.Println("Connection replaced by another client.")

	case *events.Message:
		log.Printf(
			"Message event ID=%s from=%s chat=%s",
			v.Info.ID,
			v.Info.Sender.String(),
			v.Info.Chat.String(),
			v.Info.AddressingMode,			
		)
	}
}