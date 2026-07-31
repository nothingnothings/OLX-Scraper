package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"
	"go.mau.fi/whatsmeow"
	"github.com/streadway/amqp"
)

var noPhone string

func myUsage() {
	fmt.Printf("\nWA Send Message Text Service\n\n")
	fmt.Printf("Usage: %s [OPTIONS]\n", os.Args[0])
	flag.PrintDefaults()
}

type Message struct {
	Target  string `json:"Target"`
	Message string `json:"Message"`
	Image   string `json:"Image"`
}

func main() {
	server := flag.String(
		"server",
		"amqp://"+
			os.Getenv("RABBITMQ_DEFAULT_USER")+":"+
			os.Getenv("RABBITMQ_DEFAULT_PASS")+"@"+
			os.Getenv("RABBITMQ_SERVER")+
			os.Getenv("RABBITMQ_DEFAULT_VHOST"),
		"RabbitMQ server",
	)

	queue := flag.String(
		"queue",
		os.Getenv("RABBITMQ_DEFAULT_QUEUE"),
		"RabbitMQ queue",
	)

	phone := flag.String(
		"phone",
		os.Getenv("MASTER_PHONE_NUMBER"),
		"Master phone",
	)

	flag.Usage = myUsage
	flag.Parse()

	noPhone = *phone

	conn, err := amqp.Dial(*server)
	failOnError(err, "rabbitmq")

	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "channel")

	defer ch.Close()

	q, err := ch.QueueDeclare(
		*queue,
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "consume")

	var client *whatsmeow.Client
	
	if client != nil {
		client.Disconnect()
	}
	
	client, err = connectionWhatsApp()
	
	failOnError(err, "whatsapp")

	log.Println("Waiting for RabbitMQ messages...")

	for delivery := range msgs {

		time.Sleep(5 * time.Second)

		var msg Message

		if err := json.Unmarshal(delivery.Body, &msg); err != nil {
			log.Printf("invalid message: %v", err)
			continue
		}

		if !messageCheck(msg.Target) {
			log.Printf("invalid target: %s", msg.Target)
			continue
		}

		if err := sendMessage(
			client,
			msg.Target,
			msg.Message,
			msg.Image,
		); err != nil {

			log.Printf("send failed: %v", err)
			continue
		}
	}
}

func messageCheck(str string) bool {
	re := regexp.MustCompile(`.*@g\.us|.*@s\.whatsapp\.net`)
	return re.MatchString(str)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}