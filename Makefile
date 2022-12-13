purge:
	aws sqs purge-queue --queue-url "http://localhost:4566/000000000000/myQueue.fifo" --endpoint-url http://localhost:4566

update:
	GOARCH=amd64 GOOS=linux go build -o handler
	zip handler.zip ./handler
	aws lambda update-function-code --zip-file fileb://handler.zip --function-name myFunc --endpoint-url=http://localhost:4566 

logs:
	aws --endpoint-url=http://localhost:4566 logs tail /aws/lambda/myFunc --follow

queue:
	aws sqs get-queue-attributes --queue-url "http://localhost:4566/000000000000/myQueue.fifo" --attribute-names All --endpoint-url http://localhost:4566

scan:
	aws dynamodb scan --table-name=users  --endpoint-url http://localhost:4566
