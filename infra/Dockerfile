ARG GO_VERSION
ARG NODE_VERSION
ARG PNPM_VERSION

FROM golang:${GO_VERSION} AS go-base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY internal ./internal

FROM go-base AS spec

RUN go run . openapi > /openapi.json

FROM go-base AS go-build

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/fin .

FROM node:${NODE_VERSION} AS web

ARG PNPM_VERSION

ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME/bin:$PNPM_HOME:$PATH

RUN curl -fsSL https://get.pnpm.io/install.sh | env PNPM_VERSION=${PNPM_VERSION} SHELL=bash bash -

WORKDIR /src/js

COPY js/package.json js/pnpm-lock.yaml js/.pnpmfile.mjs js/pnpm-workspace.yam[l] ./
RUN pnpm install --frozen-lockfile

COPY js/ ./
COPY --from=spec /openapi.json /src/openapi.json

RUN pnpm build

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=go-build /out/fin /app/fin
COPY --from=web /src/js/dist /app/static

ENV STATIC_DIR=/app/static

EXPOSE 8080

ENTRYPOINT ["/app/fin"]
CMD ["serve"]
