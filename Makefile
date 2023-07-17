all: test run

test:
	(cd tests && GO111MODULE=off go test -failfast)

run:
	GO111MODULE=off go run *.go --recreate

start:
	GO111MODULE=off go run *.go

mysql:
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD=password --network host \
		--name mysql mysql

mysql-client:
	docker run -ti --rm --network host mysql \
		mysql --port 3306 --password=password --protocol tcp
