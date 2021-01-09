package queue

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// AWSQueue implementation
type AWSQueue struct {
	QueueURL string
	queue    *sqs.SQS
}

// InitAWSQueue ...
func InitAWSQueue(cfg Config) Client {
	ssn := session.New(&aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewSharedCredentials(cfg.CredentialsFile, cfg.CredentialsProfile),
		MaxRetries:  aws.Int(cfg.Retries),
	})
	queue := sqs.New(ssn)
	URL := fmt.Sprintf("%s/%s", cfg.URL, cfg.Name)
	return &AWSQueue{
		queue:    queue,
		QueueURL: URL,
	}
}

// SendMessage ...
func (q AWSQueue) SendMessage(message string) error {
	// Send message
	msg := &sqs.SendMessageInput{
		MessageBody:  aws.String(message),    // Required
		QueueUrl:     aws.String(q.QueueURL), // Required
		DelaySeconds: aws.Int64(0),           // (optional) 0s - 900s (15 minutes)
	}
	sendResponse, err := q.queue.SendMessage(msg)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"event": "send_message",
		"queue": "aws_sqs",
	}).Debug(*sendResponse.MessageId)
	return nil

}

// ReceiveMessage ...
func (q AWSQueue) ReceiveMessage() (*RecvMessage, error) {
	// Receive message
	receivedMsg := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(q.QueueURL),
		MaxNumberOfMessages: aws.Int64(1),
		WaitTimeSeconds:     aws.Int64(5),
	}
	receiveResponse, err := q.queue.ReceiveMessage(receivedMsg)
	if err != nil {
		return nil, err
	}
	if len(receiveResponse.Messages) == 0 {
		return nil, errors.New("no message received")
	}
	msg := &RecvMessage{
		ID:      *receiveResponse.Messages[0].MessageId,
		Body:    *receiveResponse.Messages[0].Body,
		Handler: *receiveResponse.Messages[0].ReceiptHandle,
	}
	log.WithFields(log.Fields{
		"event": "receive_message",
		"queue": "aws_sqs",
	}).Debug(msg.ID)
	return msg, nil
}

// Acknowledge ...
func (q AWSQueue) Acknowledge(message *RecvMessage) error {
	deleteMsg := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.QueueURL),
		ReceiptHandle: &message.Handler,
	}
	_, err := q.queue.DeleteMessage(deleteMsg)
	time.Sleep(time.Second)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"event": "delete_message",
		"queue": "aws_sqs",
	}).Debug(message.ID)
	return nil
}
