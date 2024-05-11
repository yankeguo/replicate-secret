FROM golang:1.22 AS builder
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -o /replicate-secret

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /replicate-secret /replicate-secret
CMD ["/replicate-secret"]
