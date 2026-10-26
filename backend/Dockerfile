FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app/backend
COPY --from=build /out/api /app/api
COPY quality/data/ecommerce_agent_dataset /app/quality/data/ecommerce_agent_dataset
ENV API_ADDR=:8081
EXPOSE 8081
ENTRYPOINT ["/app/api"]
