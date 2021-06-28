FROM docker.io/golang:1.16 as builder

WORKDIR /workspace

COPY Makefile .
RUN make controller-gen

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

RUN make test
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY LICENSE .
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
