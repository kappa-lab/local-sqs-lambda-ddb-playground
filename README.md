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
--queue-name myQueue \
--endpoint-url http://localhost:4566
```
Check
```sh
aws sqs list-queues --endpoint-url http://localhost:4566
```
```json
{
    "QueueUrls": [
        "http://localhost:4566/000000000000/myQueue"
    ]
}
```

## 3rd Create Golang Program

```shell
go run . 
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
            }
        }
    ],
    "Count": 1,
    "ScannedCount": 1,
    "ConsumedCapacity": null
}
```
