FROM golang:alpine
RUN apk add make

WORKDIR /go/src/
COPY . /go/src/

# Erase executables that were copied from the COPY cmd above
RUN find . -maxdepth 1 -type f -executable -exec rm {} \;

# Force static builds
ENV GO_EXTLINK_ENABLED=0
ENV CGO_ENABLED=0
ENV BUILDFLAGS -ldflags \"-w -extldflags -static\" \
	-tags netgo -installsuffix netgo

RUN make cmds

FROM scratch
# FROM mysql
# ENV MYSQL_ROOT_PASSWORD=password

COPY --from=0 /go/src/server /server
COPY --from=0 /go/src/xr /xr
COPY repo.tar /

CMD [ "/server" ]
# ENTRYPOINT [ "/usr/bin/sh" ]
# CMD [ "-c", "docker-entrypoint.sh mysqld & sleep 10 && /server" ]
