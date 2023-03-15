FROM golang:alpine AS builder
LABEL maintainer="Mikolaj Gasior"

RUN apk add --update git bash openssh make gcc musl-dev

WORKDIR /go/src/Cardinal-Cryptography/docker-github-workflows-validator
COPY . .
RUN go build

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /bin
COPY --from=builder /go/src/Cardinal-Cryptography/docker-github-workflows-validator/docker-github-workflows-validator github-workflows-validator
RUN chmod +x /bin/github-workflows-validator
RUN /bin/github-workflows-validator
ENTRYPOINT ["/bin/github-workflows-validator"]
