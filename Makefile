all: run

export GO111MODULE=off

test: server
	go clean -testcache
	go test -failfast github.com/duglin/xreg-github/tests

server: *.go registry/*
	go build -o $@ .

run: server test
	./server --recreate

start: server
	./server

mysql:
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD=password --network host \
		--name mysql mysql

mysql-client:
	docker run -ti --rm --network host mysql \
		mysql --port 3306 --password=password --protocol tcp

clean:
	rm -f server
	go clean -cache -testcache
