package rabbit

import (
	"github.com/streadway/amqp"
	"fmt"
	"os"
)

func User() string{
	return os.Getenv("RABBITMQ_USER")
}

func Pwd() string{
	return os.Getenv("RABBITMQ_PASSWORD")
}

func Host() string{
	return os.Getenv("RABBITMQ_HOST")
}

func Port() string{
	return os.Getenv("RABBITMQ_PORT")
}

func Queue() string{
	return os.Getenv("RABBITMQ_QUEUE")
}


func ConnectToRabbit() *amqp.Connection {

	connection, err := amqp.Dial("amqp://" + User() +":" + Pwd() + "@" + Host() + ":" + Port() 	+ "/")
	FailOnError(err, "Failed to connect to RabbitMQ")
	return connection
}

func CreateChannel(connection *amqp.Connection) *amqp.Channel {
	rc, err := connection.Channel()
	FailOnError(err, "Failed to open a channel")
	return rc
}

func FailOnError(err error, msg string) {
	if err != nil {
	  fmt.Println("%s: %s", msg, err)
	}
}