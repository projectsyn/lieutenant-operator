FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY LICENSE .
COPY lieutenant-operator .
USER 65532:65532

ENTRYPOINT ["/lieutenant-operator"]
