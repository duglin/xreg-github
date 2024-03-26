all: mysql cmds test image run

TESTDIRS := $(shell find . -name *_test.go -exec dirname {} \; | sort -u)
IMAGE := duglin/xreg-server

cmds: server xr

qtest: .test

test: export TESTING=1
test: .test .testimage
.test: cmds */*test.go
	@echo
	@echo "# Testing"
	@go clean -testcache
	@echo "go test -failfast $(TESTDIRS)"
	@for s in $(TESTDIRS); do if ! go test -failfast $$s; then exit 1; fi; done
	@# go test -failfast $(TESTDIRS)
	@echo
	@echo "# Run again w/o cache and w/o deleting the Registry after each one"
	@go clean -testcache
	NO_CACHE=1 NO_DELETE_REGISTRY=1 go test -failfast $(TESTDIRS)
	@touch .test

unittest:
	go test -failfast ./registry

server: cmds/server.go cmds/loader.go registry/*
	@echo
	@echo "# Building server"
	go build $(BUILDFLAGS) -o $@ cmds/server.go cmds/loader.go

xr: cmds/xr.go registry/*
	@echo
	@echo "# Building CLI"
	go build $(BUILDFLAGS) -o $@ cmds/xr.go

image: .image
.image: server misc/Dockerfile
	@echo
	@echo "# Building the container image"
	@misc/errOutput docker build -f misc/Dockerfile -t $(IMAGE) --no-cache .
	@touch .image

testimage: .testimage
.testimage: .image
	@echo
	@echo "# Verifying the image"
	@misc/errOutput docker run -ti --network host $(IMAGE) --recreate --verify
	@touch .testimage

push: .push
.push: .image
	docker push $(IMAGE)
	@touch .push

notest run: mysql server image local

start: mysql server image
	@echo
	@echo "# Starting server"
	./server
	@#docker run -ti --network host $(IMAGE)

local: mysql server
	@echo
	@echo "# Starting server locally from scratch"
	./server --recreate

docker: mysql image
	@echo
	@echo "# Starting server in Docker from scratch"
	docker run -ti --network host $(IMAGE) --recreate

mysql:
	@docker container inspect mysql > /dev/null 2>&1 || \
	(echo "# Starting mysql" && \
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD=password --network host \
		--name mysql mysql > /dev/null )

mysql-client: mysql
	@while ! nc -z localhost 3306 ; do echo "Waiting for mysql" ; sleep 2 ; done
	@(docker container inspect mysql-client > /dev/null 2>&1 && \
		echo "Attaching to existing client... (press enter for prompt)" && \
		docker attach mysql-client) || \
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
	@-docker rm -f mysql mysql-client > /dev/null 2>&1
	@docker system prune -f > /dev/null
