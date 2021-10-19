FROM golang:1.17.2-alpine AS builder

ENV GO111MODULE=on
ENV APP_HOME /go/src/ecsproxy

RUN apk update
RUN mkdir -p $APP_HOME
WORKDIR $APP_HOME
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o ecsproxy

FROM nginx:1.21-alpine AS proxy
LABEL version=0.0.1
LABEL name=ecsproxy

WORKDIR /root/
ENV APP_HOME_BUILDER /go/src/ecsproxy

COPY --from=builder $APP_HOME_BUILDER/ecsproxy /usr/local/bin/
COPY template.tmpl .
COPY init.sh .

ENTRYPOINT ["./init.sh"]
