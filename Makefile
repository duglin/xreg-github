all: run

TESTDIRS := $(shell find . -name *_test.go -exec dirname {} \; | sort -u)
IMAGE := duglin/xreg-server

test: .test
.test: server */*test.go
	@go clean -testcache
	for s in $(TESTDIRS); do if ! go test -failfast $$s; then exit 1; fi; done
	# go test -failfast $(TESTDIRS)
	@echo
	@echo "# Run again w/o cache and deleting the Registry after each one"
	@go clean -testcache
	NO_CACHE=1 NO_DELETE_REGISTRY=1 go test -failfast $(TESTDIRS)
	@echo
	@touch .test

unittest:
	go test -failfast ./registry

server: *.go registry/*
	go build -o $@ .

image: .xreg-server
.xreg-server: server Dockerfile
	docker build -f Dockerfile -t $(IMAGE) . --network host --no-cache
	@touch .xreg-server

push: .push
.push: .xreg-server
	docker push $(IMAGE)
	@touch .push

run: mysql server test
	./server --recreate

start: mysql server
	./server

notest: mysql server
	./server --recreate

mysql:
	@docker container inspect mysql > /dev/null 2>&1 || \
	(echo "Starting mysql..." && \
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD=password --network host \
		--name mysql mysql )

mysql-client: mysql
	docker run -ti --rm --network host mysql \
		mysql --port 3306 --password=password --protocol tcp

k3d:
	@k3d cluster list | grep xreg > /dev/null || \
		(creating k3d cluster... || \
		k3d cluster create xreg --wait \
			-p 3306:32002@loadbalancer  \
			-p 8080:32000@loadbalancer ; \
		while ((kubectl get nodes 2>&1 || true ) | \
		grep -e "E0727" -e "forbidden" > /dev/null 2>&1  ) ; \
		do echo -n . ; sleep 1 ; done ; \
		kubectl apply -f mysql.yaml )

k3dserver: k3d image
	-kubectl delete -f deploy.yaml 2> /dev/null
	k3d image import duglin/xreg-server -c xreg
	kubectl apply -f deploy.yaml
	sleep 2 ; kubectl logs -f xreg-server

clean:
	rm -f server
	rm -f .test .xreg-server .push
	go clean -cache -testcache
	-k3d cluster delete xreg
	-docker rm -f mysql
