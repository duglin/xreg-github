all: run

export GO111MODULE=off

test:
	go clean -testcache
	go test -failfast github.com/duglin/xreg-github/tests

run: test
	go run . --recreate

start: test
	go run .

mysql:
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD=password --network host \
		--name mysql mysql

mysql-client:
	docker run -ti --rm --network host mysql \
		mysql --port 3306 --password=password --protocol tcp

clean:
	go clean -cache -testcache
