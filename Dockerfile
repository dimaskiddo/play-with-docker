# Builder Image
# ---------------------------------------------------
FROM golang:1.25-alpine AS go-builder

WORKDIR /usr/src/app

COPY . ./

RUN apk --no-cache --update upgrade \
    && apk --no-cache --update add \
        openssh-keygen \
        openssh-client \
    && ssh-keygen -N "" -t rsa -b 4096 \
        -f /etc/ssh/ssh_host_rsa_key \
        -C docker@dimaskiddo.my.id > /dev/null \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix nocgo -o play-with-docker .


# Final Image
# ---------------------------------------------------
FROM dimaskiddo/alpine:base-glibc
MAINTAINER Dimas Restu Hidayanto <dimas.restu@student.upi.edu>

WORKDIR /usr/app/play-with-docker

RUN apk --no-cache --update upgrade \
    && apk --no-cache --update add \
        ca-certificates \
        openssh-client \
    && mkdir -p /usr/app/play-with-docker/pwd

COPY --from=go-builder /etc/ssh/ssh_host_rsa_key /etc/ssh/ssh_host_rsa_key
COPY --from=go-builder /usr/src/app/play-with-docker /usr/bin/play-with-docker

EXPOSE 3000

CMD ["play-with-docker"]
