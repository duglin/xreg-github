FROM golang:alpine
RUN apk add git
WORKDIR /go/src/
COPY . /go/src/
RUN go get -d github.com/duglin/xreg-github
RUN GO_EXTLINK_ENABLED=0 CGO_ENABLED=0 go build \
    -ldflags "-w -extldflags -static" \
	-tags netgo -installsuffix netgo \
	-o /server github.com/duglin/xreg-github

FROM scratch
COPY --from=0 /server /server
COPY repo.tar /
CMD [ "/server" ]
