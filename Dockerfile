# syntax=docker/dockerfile:1
ARG app_dir="/home/go/app"

FROM golang:1.25-alpine3.22 AS build
ARG app_dir
WORKDIR ${app_dir}

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod/ \
	CGO_ENABLED=0 go build -v -o ${app_dir}/build/server .

FROM alpine:3.22 AS final
ARG app_dir
WORKDIR ${app_dir}

RUN addgroup go && adduser -D -G go go
RUN mkdir -p ${app_dir}/log ${app_dir}/reports && chown -R go:go ${app_dir}
USER go

COPY --from=build ${app_dir}/build/server ${app_dir}/server
CMD [ "./server" ]
