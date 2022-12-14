Run
```
docker-compose up -d
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