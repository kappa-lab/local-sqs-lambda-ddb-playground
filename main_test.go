package main_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type User struct {
	User_id   string
	User_name string
}

func Test_Duplicate_Check(t *testing.T) {
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: "http://localhost:4566"}, nil
			})))
	require.NoError(t, err)

	sqsClient := sqs.NewFromConfig(conf)
	createOut, err := sqsClient.CreateQueue(ctx,
		&sqs.CreateQueueInput{
			QueueName: aws.String("test.fifo"),
			Attributes: map[string]string{
				"FifoQueue":                 "true",
				"DeduplicationScope":        "queue",
				"FifoThroughputLimit":       "perQueue",
				"ContentBasedDeduplication": "true",
			},
		},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := sqsClient.DeleteQueue(ctx,
			&sqs.DeleteQueueInput{QueueUrl: createOut.QueueUrl},
		)
		require.NoError(t, err)
	})

	_, err = sqsClient.SendMessage(ctx,
		&sqs.SendMessageInput{
			QueueUrl:       createOut.QueueUrl,
			MessageBody:    aws.String("msg1"),
			MessageGroupId: aws.String("g1"),
		},
	)
	require.NoError(t, err)

	_, err = sqsClient.SendMessage(ctx,
		&sqs.SendMessageInput{
			QueueUrl:       createOut.QueueUrl,
			MessageBody:    aws.String("msg1"),
			MessageGroupId: aws.String("g1"),
		},
	)
	require.NoError(t, err)

	attrOut, err := sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: createOut.QueueUrl,
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
			types.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
		},
	},
	)
	require.NoError(t, err)
	assert.Equal(t, "1", attrOut.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessages)])
	assert.Equal(t, "0", attrOut.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessagesNotVisible)])
}

func Test_FIFO_RoundTrip(t *testing.T) {
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: "http://localhost:4566"}, nil
			})))
	require.NoError(t, err)

	sqsClient := sqs.NewFromConfig(conf)

	_, err = sqsClient.SendMessage(ctx,
		&sqs.SendMessageInput{
			QueueUrl:    aws.String("http://localhost:4566/000000000000/myQueue.fifo"),
			MessageBody: aws.String("user2"),
			MessageAttributes: map[string]types.MessageAttributeValue{
				"name": {
					DataType:    aws.String("String"),
					StringValue: aws.String("Cola"),
				},
			},
			MessageGroupId: aws.String("g1"),
		},
	)

	require.NoError(t, err)

	ddbClient := dynamodb.NewFromConfig(conf)
	out, err := ddbClient.Scan(ctx,
		&dynamodb.ScanInput{
			TableName: aws.String("users"),
		},
		dynamodb.WithEndpointResolver(dynamodb.EndpointResolverFromURL("http://localhost:4566")),
	)

	require.NoError(t, err)

	var items []User
	err = attributevalue.UnmarshalListOfMaps(out.Items, &items)
	require.NoError(t, err)

	assert.Equal(t, User{User_id: "user2", User_name: "Cola"}, items[0])
}
