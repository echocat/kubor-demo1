FROM golang:1.12.1 AS builder

ARG BRANCH=development
ARG REVISION=latest
COPY  . /src
WORKDIR /src
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -o /tmp/app -ldflags "-X main.branch=$BRANCH main.revision=$REVISION" app.go

FROM scratch
COPY --from=builder /tmp/app /app
USER 5000
ENTRYPOINT ["/app"]
