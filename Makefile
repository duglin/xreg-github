all: run

export GO111MODULE=off

test: server
	@go clean -testcache
	go test -failfast github.com/duglin/xreg-github/tests
	@echo
	@echo "# Run the tests again w/o deleting the Registry after each one"
	@go clean -testcache
	NO_DELETE_REGISTRY=1 go test -failfast github.com/duglin/xreg-github/tests
	@echo

server: *.go registry/*
	go build -o $@ .

run: server test
	./server --recreate

start: server
	./server

notest: server
	./server --recreate

mysql:
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD=password --network host \
		--name mysql mysql

mysql-client:
	docker run -ti --rm --network host mysql \
		mysql --port 3306 --password=password --protocol tcp

clean:
	rm -f server
	go clean -cache -testcache
