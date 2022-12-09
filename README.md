Run
```
docker-compose up -d
```


# 1st DDB 
## Create Table
```
aws dynamodb create-table --cli-input-json file://user.json --endpoint-url http://localhost:4566
```

## Show Table
```
aws dynamodb list-tables  --endpoint-url http://localhost:4566
```

```
{
    "TableNames": [
        "users"
    ]
}
```
