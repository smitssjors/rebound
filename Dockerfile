FROM golang:1.23 AS build

WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -o /go/bin/app

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /go/bin/app /
EXPOSE 3000
CMD ["/app"]
