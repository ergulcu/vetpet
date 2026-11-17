# Copyright 2019 Mehmet ERGULCU
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/vetpet ./cmd/vetpet && mkdir /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/vetpet /vetpet
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV VETPET_DATABASE=/data/vetpet.db
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/vetpet"]
