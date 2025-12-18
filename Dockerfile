# Builder Image
# ---------------------------------------------------
FROM golang:1.25-alpine AS builder

WORKDIR /usr/src/app

COPY . ./

RUN apk --no-cache --update upgrade \
    && apk --no-cache --update add \
        openssh-keygen \
    && ssh-keygen -N "" -t rsa -b 4096 \
        -f ./ssh_host_rsa_key \
        -C docker@dimaskiddo.my.id > /dev/null \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix nocgo -o play-with-docker . \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix nocgo -o play-with-docker-router ./router/l2


# Final Image (Base)
# ---------------------------------------------------
FROM dimaskiddo/alpine:base-glibc AS base
MAINTAINER Dimas Restu Hidayanto <dimas.restu@student.upi.edu>

WORKDIR /usr/app/play-with-docker

RUN apk --no-cache --update upgrade \
    && apk --no-cache --update add \
        ca-certificates \
    && mkdir -p \
        /usr/app/play-with-docker/certs \
        /usr/app/play-with-docker/sessions

COPY --from=builder /usr/src/app/ssh_host_rsa_key ./ssh_host_rsa_key


# Final Image (Play-With-Docker)
# ---------------------------------------------------
FROM base AS play-with-docker

COPY --from=builder /usr/src/app/play-with-docker /usr/bin/play-with-docker

EXPOSE 3000

CMD ["play-with-docker"]


# Final Image (Play-With-Docker Router)
# ---------------------------------------------------
FROM base AS play-with-docker-router

COPY --from=builder /usr/src/app/play-with-docker-router /usr/bin/play-with-docker-router

EXPOSE 22 53 443

CMD ["play-with-docker-router"]
