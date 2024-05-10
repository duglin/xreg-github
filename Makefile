all: mysql cmds test image run

# Notes:
# export XR_SPEC=$HOME/go/src/github.com/xregistry/spec -> to load local models
# export VERBOSE=[0-9]                                  -> control log verbosity
# Override these env vars as needed:
DBHOST     ?= 127.0.0.1
DBPORT     ?= 3306
DBUSER     ?= root
DBPASSWORD ?= password
IMAGE      ?= duglin/xreg-server

TESTDIRS := $(shell find . -name *_test.go -exec dirname {} \; | sort -u)

ifdef XR_SPEC
  # If pointing to local spec then make sure "docker run" uses it too
  DOCKER_SPEC=/spec
endif

cmds: .cmds
.cmds: server xr
	@touch .cmds

qtest: .test

test: .test .testimage
.test: export TESTING=1
.test: .cmds */*test.go
	@make --no-print-directory mysql waitformysql
	@echo
	@echo "# Testing"
	@go clean -testcache
	@echo "go test -failfast $(TESTDIRS)"
	@for s in $(TESTDIRS); do if ! go test -failfast $$s; then exit 1; fi; done
	@# go test -failfast $(TESTDIRS)
	@echo
	@echo "# Run again w/o deleting the Registry after each one"
	@go clean -testcache
	NO_DELETE_REGISTRY=1 go test -failfast $(TESTDIRS)
	@touch .test

unittest:
	go test -failfast ./registry

server: cmds/server.go cmds/loader.go registry/*
	@echo
	@echo "# Building server"
	go build $(BUILDFLAGS) -o $@ cmds/server.go cmds/loader.go

xr: cmds/xr*.go registry/*
	@echo
	@echo "# Building CLI"
	go build $(BUILDFLAGS) -o $@ cmds/xr*.go

image: .image
.image: server misc/Dockerfile misc/waitformysql misc/Dockerfile-all \
		misc/startall
	@echo
	@echo "# Building the container image"
ifdef XR_SPEC
	# Copy local xReg spec files into tmp dir that "docker build" looks for
	@rm -rf .spec
	@mkdir -p .spec
	cp -r $(XR_SPEC)/* .spec
endif
	@misc/errOutput docker build -f misc/Dockerfile -t $(IMAGE) --no-cache .
	@misc/errOutput docker build -f misc/Dockerfile-all -t $(IMAGE)-all \
		--no-cache .
ifdef XR_SPEC
	@rm -rf .spec
endif
	@touch .image

testimage: .testimage
.testimage: .image
	@echo
	@echo "# Verifying the image"
	@make --no-print-directory mysql waitformysql
	@misc/errOutput docker run -ti \
		-e DBHOST=$(DBHOST) -e DBPORT=$(DBPORT) -e DBUSER=$(DBUSER) \
		-e XR_SPEC=$(DOCKER_SPEC) \
		--network host \
		$(IMAGE) --recreate --verify
	@touch .testimage

push: .push
.push: .image
	docker push $(IMAGE)
	docker push $(IMAGE)-all
	@touch .push

notest run: mysql server local

start: mysql server waitformysql #image
	@echo
	@echo "# Starting server"
	./server
	@#docker run -ti --network host $(IMAGE)

local: mysql server waitformysql
	@echo
	@echo "# Starting server locally from scratch"
	./server --recreate

docker-all: image
	docker run -ti -p 8080:8080 $(IMAGE)-all --recreate

large:
	@XR_LOAD_LARGE=1 make --no-print-directory run

docker: mysql image waitformysql
	@echo
	@echo "# Starting server in Docker from scratch"
	docker run -ti --network host $(IMAGE) --recreate

mysql:
	@docker container inspect mysql > /dev/null 2>&1 || \
	(echo "# Starting mysql" && \
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD="$(DBPASSWORD)" \
		-p $(DBPORT):$(DBPORT) --name mysql mysql > /dev/null )
		@ # -e MYSQL_USER=$(DBUSER) \

waitformysql:
	@while ! docker run -ti --network host mysql mysqladmin \
		-h $(DBHOST) -P $(DBPORT) -s ping ;\
	do \
		echo "Waiting for mysql" ; \
		sleep 2 ; \
	done

mysql-client: mysql waitformysql
	@(docker container inspect mysql-client > /dev/null 2>&1 && \
		echo "Attaching to existing client... (press enter for prompt)" && \
		docker attach mysql-client) || \
	docker run -ti --rm --network host --name mysql-client mysql \
		mysql --host $(DBHOST) --port $(DBPORT) \
		--user $(DBUSER) --password="$(DBPASSWORD)" \
		--protocol tcp || \
		echo "If it failed, make sure mysql is ready"

k3d: misc/mysql.yaml
	@k3d cluster list | grep xreg > /dev/null || \
		(creating k3d cluster || \
		k3d cluster create xreg --wait \
			-p $(DBPORT):32002@loadbalancer  \
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

prof: server qtest
	@# May need to install: apt-get install graphviz
	NO_DELETE_REGISTRY=1 \
		go test -cpuprofile cpu.prof -memprofile mem.prof -bench . \
		github.com/duglin/xreg-github/tests
	@# go tool pprof -http:0.0.0.0:9999 cpu.prof
	@go tool pprof -top -cum cpu.prof | sed -n '0,/flat/p;/xreg/p' | more
	@rm -f cpu.prof mem.prof tests.test

clean:
	@echo "# Cleaning"
	@rm -f cpu.prof mem.prof
	@rm -f server xr
	@rm -f .test .image .push
	@go clean -cache -testcache
	@-k3d cluster delete xreg > /dev/null 2>&1
	@-docker rm -f mysql mysql-client > /dev/null 2>&1
	@docker system prune -f > /dev/null
