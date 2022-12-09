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
