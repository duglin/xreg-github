all: run

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

image: server Dockerfile
	docker build -f Dockerfile -t duglin/xreg-server .

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
	go clean -cache -testcache
	k3d cluster delete xreg
