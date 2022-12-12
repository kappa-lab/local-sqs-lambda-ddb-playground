package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const tableName = "users"

func main() {
	isStandAlone := flag.Bool("isStandAlone", false, "default is false")
	flag.Parse()

	if *isStandAlone {
		_ = createItem("u1", "tea")
		return
	}
	lambda.Start(handler)
}

func handler(ctx context.Context, evt events.SQSEvent) {
	name := ""
	if nameAttr, has := evt.Records[0].MessageAttributes["name"]; has {
		name = *nameAttr.StringValue
	}
	err := createItem(evt.Records[0].Body, name)

	if err != nil {
		log.Fatal(err)
	}
}

func createItem(userId string, userName string) error {
	log.Printf("[userId:%s , userName:%s]", userId, userName)
	if userName == "" {
		return fmt.Errorf("Name must not empty")
	}
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	ddbClient := dynamodb.NewFromConfig(conf)

	out, err := ddbClient.ListTables(ctx,
		&dynamodb.ListTablesInput{
			Limit: aws.Int32(100),
		},
		dynamodb.WithEndpointResolver(dynamodb.EndpointResolverFromURL("http://localhost:4566")),
	)

	if err != nil {
		return err
	}

	if len(out.TableNames) == 0 && out.TableNames[0] != tableName {
		return fmt.Errorf("table not found:%s", out.TableNames)
	}

	_, err = ddbClient.PutItem(ctx,
		&dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item: map[string]types.AttributeValue{
				"user_id":   &types.AttributeValueMemberS{Value: userId},
				"user_name": &types.AttributeValueMemberS{Value: userName},
			},
		},
		dynamodb.WithEndpointResolver(dynamodb.EndpointResolverFromURL("http://localhost:4566")),
	)
	if err != nil {
		return err
	}
	return nil
}
