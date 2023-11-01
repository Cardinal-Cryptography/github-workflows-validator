FROM golang:alpine AS builder
LABEL maintainer="devops@alephzero"

RUN apk add --update git bash openssh make gcc musl-dev

WORKDIR /go/src/Cardinal-Cryptography/github-actions-validator
COPY . .
RUN go build

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /bin
COPY --from=builder /go/src/Cardinal-Cryptography/github-actions-validator/github-actions-validator github-actions-validator
RUN chmod +x /bin/github-actions-validator
RUN /bin/github-actions-validator
ENTRYPOINT ["/bin/github-actions-validator"]
