package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const tableName = "users"

func main() {
	createItem("u1")
}
func createItem(userId string) {
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	ddbClient := dynamodb.NewFromConfig(conf)

	out, err := ddbClient.ListTables(ctx,
		&dynamodb.ListTablesInput{
			Limit: aws.Int32(100),
		},
		dynamodb.WithEndpointResolver(dynamodb.EndpointResolverFromURL("http://localhost:4566")),
	)

	if err != nil {
		log.Fatal(err)
	}

	if len(out.TableNames) == 0 && out.TableNames[0] != tableName {
		log.Fatal("table not found:", out.TableNames)
	}

	_, err = ddbClient.PutItem(ctx,
		&dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item: map[string]types.AttributeValue{
				"user_id": &types.AttributeValueMemberS{Value: userId},
			},
		},
		dynamodb.WithEndpointResolver(dynamodb.EndpointResolverFromURL("http://localhost:4566")),
	)
	if err != nil {
		log.Fatal(err)
	}
}
