# 概要

LocalStackを使って、ローカル環境でSQS → Lambda → DynamoDB
の連携をテストするためのサンプルです。

## 動作フロー

1. SQS (FIFO Queue) にメッセージを送信
2. Lambda関数 がSQSのメッセージをトリガーとして起動
3. Lambda がメッセージ内容を DynamoDB の users テーブルに保存

## 主な手順

1. LocalStackをDocker Composeで起動
2. DynamoDBテーブル作成 (users)
3. SQS FIFO キュー作成 (myQueue.fifo)
4. GoプログラムをビルドしてLambda関数としてデプロイ
5. SQSとLambdaをEvent Source Mappingで連携
6. SQSにメッセージを送信するとLambdaが実行され、DDBにレコードが追加される

# Setup
Run
```
docker-compose up -d
```

AWS Profileが設定されていない場合は、環境変数を設定してください。
```
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=ap-northeast-1
```

# 1st DDB 
## Create Table
```shell
aws dynamodb create-table \
--cli-input-json file://user.json \
--endpoint-url http://localhost:4566
```

Check
```shell
aws dynamodb list-tables  --endpoint-url http://localhost:4566
```
```json
{
    "TableNames": [
        "users"
    ]
}
```

# 2nd SQS
## Create Queue
```shell
aws sqs create-queue \
--cli-input-json file://myQueue.fifo.json \
--endpoint-url http://localhost:4566
```
Check
```sh
aws sqs list-queues --endpoint-url http://localhost:4566
```
```json
{
    "QueueUrls": [
        "http://sqs.ap-northeast-1.localhost.localstack.cloud:4566/000000000000/myQueue.fifo"
    ]
}
```

補足: 旧バージョンは以下を返していた。現行バージョンではより本番に近いURLを返すようになった。ただしlocalhostでアクセスできる。

```json
{
    "QueueUrls": [
        "http://localhost:4566/000000000000/myQueue.fifo"
    ]
}
```

## 3rd Create Golang Program

```shell
go run . -isStandAlone=true 
```

Check
```shell
aws dynamodb scan --table-name=users  --endpoint-url http://localhost:4566
```
```json
{
    "Items": [
        {
            "user_id": {
                "S": "u1"
            },
            "user_name": {
                "S": "tea"
            }
        }
    ],
    "Count": 1,
    "ScannedCount": 1,
    "ConsumedCapacity": null
}
```

## 4th Integration Lambda
### Build App
```sh
GOARCH=amd64 GOOS=linux go build -o handler
zip handler.zip ./handler
```

### Create Function
```sh
aws lambda create-function \
--zip-file fileb://handler.zip \
--function-name myFunc \
--runtime go1.x \
--role test \
--handler handler \
--endpoint-url=http://localhost:4566 
```
### Apply SQS Event Source
Createでは設定できない
```
aws --endpoint-url=http://localhost:4566 \
lambda create-event-source-mapping \
--function-name myFunc \
--event-source-arn arn:aws:sqs:ap-northeast-1:000000000000:myQueue.fifo
```

### Update Function
実装を修正した場合に実行する
```sh
aws lambda update-function-code \
--zip-file fileb://handler.zip \
--function-name myFunc \
--endpoint-url=http://localhost:4566 
```
### Show Log
```sh
aws --endpoint-url=http://localhost:4566 logs tail /aws/lambda/myFunc --follow
```

### Send SQS
```shell
aws sqs send-message \
--queue-url "http://localhost:4566/000000000000/myQueue.fifo" \
--message-body user2 \
--message-group-id g1 \
--message-attributes '{"name":{"DataType":"String", "StringValue": "Coffee"}}' \
--endpoint-url=http://localhost:4566 
```
 

Check
```shell
aws dynamodb scan --table-name=users  --endpoint-url http://localhost:4566
```
```json
{
    "Items": [
        {
            "user_id": {
                "S": "user2"
            },
            "user_name": {
                "S": "Coffee"
            }
        },
        {
            "user_id": {
                "S": "u1"
            },
            "user_name": {
                "S": "tea"
            }
        }
    ],
    "Count": 2,
    "ScannedCount": 2,
    "ConsumedCapacity": null
}
```

Check
```shell
aws sqs get-queue-attributes \
--queue-url "http://localhost:4566/000000000000/myQueue.fifo" \
--attribute-names ApproximateNumberOfMessages \
--query Attributes.ApproximateNumberOfMessages \
--endpoint-url http://localhost:4566
```
```json
"0"
```