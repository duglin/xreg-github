all: mysql cmds test image run

TESTDIRS := $(shell find . -name *_test.go -exec dirname {} \; | sort -u)
IMAGE := duglin/xreg-server

cmds: server xr

test: export TESTING=1
test: .test
.test: server */*test.go
	@echo
	@echo "# Testing"
	@go clean -testcache
	for s in $(TESTDIRS); do if ! go test -failfast $$s; then exit 1; fi; done
	@# go test -failfast $(TESTDIRS)
	@echo
	@echo "# Run again w/o cache and w/o deleting the Registry after each one"
	@go clean -testcache
	NO_CACHE=1 NO_DELETE_REGISTRY=1 go test -failfast $(TESTDIRS)
	@echo
	@touch .test

unittest:
	go test -failfast ./registry

server: cmds/server.go cmds/loader.go registry/*
	@echo
	@echo "# Building server"
	go build $(BUILDFLAGS) -o $@ cmds/server.go cmds/loader.go

xr: cmds/xr.go registry/*
	@echo "# Building CLI"
	go build $(BUILDFLAGS) -o $@ cmds/xr.go

image: .image
.image: server misc/Dockerfile
	@echo docker build -f misc/Dockerfile -t $(IMAGE) --no-cache .
	@docker build -f misc/Dockerfile -t $(IMAGE) --no-cache . \
		> .dockerout 2>&1 || { cat .dockerout ; rm .dockerout ; exit 1 ; }
	@rm .dockerout
	@touch .image

push: .push
.push: .image
	docker push $(IMAGE)
	@touch .push

run: mysql server
	@echo
	./server --recreate

start: mysql server
	./server

notest: mysql server
	./server --recreate

mysql:
	@docker container inspect mysql > /dev/null 2>&1 || \
	(echo "# Starting mysql" && \
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD=password --network host \
		--name mysql mysql > /dev/null )

mysql-client: mysql
	@while ! nc -z localhost 3306 ; do echo "Waiting for mysql" ; sleep 2 ; done
	docker run -ti --rm --network host --name mysql-client mysql \
		mysql --port 3306 --password=password --protocol tcp || \
		echo "If it failed, make sure mysql is ready"

k3d: misc/mysql.yaml
	@k3d cluster list | grep xreg > /dev/null || \
		(creating k3d cluster || \
		k3d cluster create xreg --wait \
			-p 3306:32002@loadbalancer  \
			-p 8080:32000@loadbalancer ; \
		while ((kubectl get nodes 2>&1 || true ) | \
		grep -e "E0727" -e "forbidden" > /dev/null 2>&1  ) ; \
		do echo -n . ; sleep 1 ; done ; \
		kubectl apply -f misc/mysql.yaml )

k3dserver: k3d image
	-kubectl delete -f misc/deploy.yaml 2> /dev/null
	k3d image import $(IMAGE) -c xreg
	kubectl apply -f misc/deploy.yaml
	sleep 2 ; kubectl logs -f xreg-server

clean:
	@echo "# Cleaning"
	@rm -f server xr
	@rm -f .test .image .push
	@go clean -cache -testcache
	@-k3d cluster delete xreg > /dev/null 2>&1
	@-docker rm -f mysql > /dev/null 2>&1
	@docker system prune -f > /dev/null
