#build stage
FROM golang:1.10.3-alpine3.8 AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY . .
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/app .
CMD ["./app"]
LABEL Name=aksnodepublicipcontroller Version=0.2