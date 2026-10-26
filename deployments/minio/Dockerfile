FROM golang:1.24-bookworm AS build
WORKDIR /src

# The original public release images are no longer anonymously accessible.
# Build the same upstream release, verifying the pinned source archive.
RUN curl --fail --location --retry 3 \
      https://codeload.github.com/minio/minio/tar.gz/refs/tags/RELEASE.2025-04-22T22-12-26Z \
      --output /tmp/minio-source.tar.gz \
    && echo "1a63209927b9d1fb239b2a776b7b01d1902eff19379a108f4bc7005edcc0968c  /tmp/minio-source.tar.gz" | sha256sum --check \
    && tar -xzf /tmp/minio-source.tar.gz --strip-components=1 -C /src
RUN go mod download
RUN CGO_ENABLED=0 go build -trimpath -o /out/minio .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/minio /usr/bin/minio
EXPOSE 9000 9001
VOLUME ["/data"]
ENTRYPOINT ["/usr/bin/minio"]
