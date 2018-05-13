FROM golang:1.10.2-alpine3.7 AS build

WORKDIR /go/src/loe.yt/pass_server
COPY . .

RUN go install -v ./...

FROM alpine:3.7
RUN apk --no-cache add gnupg
COPY --from=build /go/bin/pass_server /usr/bin/pass_server
CMD ["pass_server"]
