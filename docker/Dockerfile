###
### Service Builder
###
FROM golang:1.24-bookworm@sha256:3ce988c30fa67dc966ca716ee0ce7ad08d7330573e808cb68ada7e419bdf23de AS service-builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make bin/userclouds

###
### UI Builder
###
FROM node:22.5.1-bookworm-slim@sha256:2fb92fe9d7350866a73c5cc311c1a19919ffd47e8592d4233374ee330e3bdb1e AS console-builder

ENV PATH=/root/.local/bin:$PATH
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

WORKDIR /userclouds

RUN : \
    && apt-get update \
    && apt install curl build-essential -y \
    && :

COPY console console
COPY plex plex
COPY sharedui sharedui
COPY ui-component-lib ui-component-lib
COPY *.lock .
COPY *.json .
COPY eslint.config.mjs .
COPY pyproject.toml .

RUN : \
    && curl -LsSf https://astral.sh/uv/install.sh | sh \
    && uv python install 3.12 \
    && npm install -g corepack \
    && yarn config set nodeLinker node-modules \
    && uv run yarn install --immutable \
    && yarn sharedui:build \
    && yarn ui-lib:build \
    && yarn consoleui:build \
    && yarn plexui:build \
    && :

###
### Release image
###
FROM debian:bookworm-slim@sha256:df52e55e3361a81ac1bead266f3373ee55d29aa50cf0975d440c2be3483d8ed3

ENV PATH=/userclouds/bin:$PATH
ENV UC_CONFIG_DIR=/userclouds/config/
ENV UC_UNIVERSE=onprem
ENV UC_REGION=customerlocal
ENV UC_CONFIG_MANIFEST_FILE=""

RUN : \
    && apt update \
    && apt install gnupg2 lsb-release ca-certificates -y \
    && :

COPY config/userclouds /userclouds/config

# Copy binaries
COPY --from=service-builder /usr/src/app/bin/. /userclouds/bin/

# Copy UI components
COPY --from=console-builder /userclouds/console/consoleui/build/* /userclouds/uiassets/console/
COPY --from=console-builder /userclouds/plex/plexui/build/* /userclouds/uiassets/plex/

RUN : \
    && groupadd --gid 99 userclouds \
    && useradd -m --no-log-init --uid 99 --gid 99 userclouds \
    && chown userclouds:userclouds /userclouds/bin \
    && chown -R userclouds:userclouds /userclouds/config \
    && chown -R userclouds:userclouds /userclouds/uiassets \
    && chmod u=r,go=r /userclouds/config/provisioning/*.json \
    && chmod u=rx,go=rx /userclouds/bin \
    && :

RUN : \
    && apt clean \
    && rm -rf /var/lib/apt/lists/* \
    && :

USER userclouds:userclouds
EXPOSE 3040
CMD ["/userclouds/bin/userclouds"]
