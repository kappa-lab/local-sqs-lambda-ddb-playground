package main_test

import (
	"context"
	"encoding/json"
	"fmt"
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

func prepareConf(t *testing.T) aws.Config {
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: "http://localhost:4566"}, nil
			})))
	require.NoError(t, err)

	return conf
}

func Test_Duplicate_Check(t *testing.T) {
	ctx := context.Background()
	sqsClient := sqs.NewFromConfig(prepareConf(t))
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
	conf := prepareConf(t)
	sqsClient := sqs.NewFromConfig(conf)

	_, err := sqsClient.SendMessage(ctx,
		&sqs.SendMessageInput{
			QueueUrl:    aws.String("http://sqs.ap-northeast-1.localhost.localstack.cloud:4566/000000000000/myQueue.fifo"),
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

	// Lambdaのセットアップが完了していることが前提
	assert.Equal(t, User{User_id: "user2", User_name: "Cola"}, items[0])
}

func Test_QUEUE_CRUD_SIMPLE(t *testing.T) {
	ctx := context.Background()
	conf := prepareConf(t)
	sqsClient := sqs.NewFromConfig(conf)

	// テスト専用のキューを作成（Lambdaと連携していないキュー）
	createOut, err := sqsClient.CreateQueue(ctx,
		&sqs.CreateQueueInput{
			QueueName: aws.String("test-crud.fifo"),
			Attributes: map[string]string{
				"FifoQueue":                 "true",
				"ContentBasedDeduplication": "true", // メッセージ本文ベースの重複排除を有効化
			},
		},
	)
	require.NoError(t, err)
	queueUrl := createOut.QueueUrl

	// テスト終了時にキューを削除
	t.Cleanup(func() {
		_, _ = sqsClient.DeleteQueue(ctx, &sqs.DeleteQueueInput{
			QueueUrl: queueUrl,
		})
	})

	// 送信 (Create)
	sendOut, err := sqsClient.SendMessage(ctx,
		&sqs.SendMessageInput{
			QueueUrl:    queueUrl,
			MessageBody: aws.String("execute-id-1"),
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
	fmt.Printf("Sent MessageId: %s\n", *sendOut.MessageId)

	// 受信 (Read)
	receiveOut, err := sqsClient.ReceiveMessage(ctx,
		&sqs.ReceiveMessageInput{
			QueueUrl:            queueUrl,
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     1,
			MessageAttributeNames: []string{
				"All",
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, receiveOut.Messages, 1)

	msg := receiveOut.Messages[0]
	// メッセージ全体をJSON形式で出力
	msgJSON, err := json.MarshalIndent(msg, "", "  ")
	require.NoError(t, err)
	fmt.Printf("Message JSON:\n%s\n", string(msgJSON))

	// 削除 (Delete)
	_, err = sqsClient.DeleteMessage(ctx,
		&sqs.DeleteMessageInput{
			QueueUrl:      queueUrl,
			ReceiptHandle: msg.ReceiptHandle,
		},
	)
	require.NoError(t, err)
	fmt.Printf("Deleted message with MessageId: %s\n", *msg.MessageId)

	// 削除確認 - もう一度受信を試みて、メッセージがないことを確認
	confirmOut, err := sqsClient.ReceiveMessage(ctx,
		&sqs.ReceiveMessageInput{
			QueueUrl:            queueUrl,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     1,
		},
	)
	require.NoError(t, err)
	assert.Zero(t, len(confirmOut.Messages), "Message should be deleted")
}

func Test_QUEUE_CRUD_Consistency(t *testing.T) {
	ctx := context.Background()
	conf := prepareConf(t)
	sqsClient := sqs.NewFromConfig(conf)

	// テスト専用のキューを作成（Lambdaと連携していないキュー）
	createOut, err := sqsClient.CreateQueue(ctx,
		&sqs.CreateQueueInput{
			QueueName: aws.String("test-crud-consistent.fifo"),
			Attributes: map[string]string{
				"FifoQueue":                 "true",
				"ContentBasedDeduplication": "true",
				"VisibilityTimeout":         "3", // 3秒経過で再度取得可能にする
			},
		},
	)
	require.NoError(t, err)
	queueUrl := createOut.QueueUrl

	// テスト終了時にキューを削除
	t.Cleanup(func() {
		_, _ = sqsClient.DeleteQueue(ctx, &sqs.DeleteQueueInput{
			QueueUrl: queueUrl,
		})
	})

	// 3件送信 (Create)するが、重複により実際は2件しか入らない
	_, err = sqsClient.SendMessage(ctx,
		&sqs.SendMessageInput{
			QueueUrl:       queueUrl,
			MessageBody:    aws.String("execute-id-1"),
			MessageGroupId: aws.String("g1"),
		},
	)
	require.NoError(t, err)
	_, err = sqsClient.SendMessage(ctx,
		&sqs.SendMessageInput{
			QueueUrl:       queueUrl,
			MessageBody:    aws.String("execute-id-2"),
			MessageGroupId: aws.String("g1"),
		},
	)
	require.NoError(t, err)
	_, err = sqsClient.SendMessage(ctx,
		&sqs.SendMessageInput{
			QueueUrl:       queueUrl,
			MessageBody:    aws.String("execute-id-2"),
			MessageGroupId: aws.String("g1"),
		},
	)
	require.NoError(t, err)

	// 受信 (Read)
	receiveOut, err := sqsClient.ReceiveMessage(ctx,
		&sqs.ReceiveMessageInput{
			QueueUrl:            queueUrl,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     1,
			MessageAttributeNames: []string{
				"All",
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, receiveOut.Messages, 2)

	// メッセージ全体をJSON形式で出力
	msgJSON, err := json.MarshalIndent(receiveOut.Messages, "", "  ")
	require.NoError(t, err)
	fmt.Printf("Message JSON:\n%s\n", string(msgJSON))

	// 前回受信から3秒経過してないので、再度受信してもメッセージは取得できない
	receiveOut, err = sqsClient.ReceiveMessage(ctx,
		&sqs.ReceiveMessageInput{
			QueueUrl:            queueUrl,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     1,
			MessageAttributeNames: []string{
				"All",
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, receiveOut.Messages, 0)

	// VisibilityTimeout(3秒)より長い時間(4秒)待機することで、
	// タイムアウトが切れた時点でメッセージが再取得可能になる。
	// time.Sleepを使わず、WaitTimeSecondsのみで制御することを確認
	receiveOut, err = sqsClient.ReceiveMessage(ctx,
		&sqs.ReceiveMessageInput{
			QueueUrl:            queueUrl,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     3 + 2, // 5秒待つ
			MessageAttributeNames: []string{
				"All",
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, receiveOut.Messages, 2)

}
