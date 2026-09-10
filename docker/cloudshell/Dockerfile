FROM node:18.16.0 as builder

# Build frontend code which added upload/download button
WORKDIR /app
COPY html/package.json /app/
COPY html/yarn.lock /app/
RUN yarn config set network-timeout 100000
RUN yarn install
COPY html/ /app/
RUN yarn run build

FROM ghcr.io/dtzar/helm-kubectl:3.19.1
SHELL [ "/bin/bash", "-c" ]

ARG TTYD_VERSION=1.7.8
ARG HELM_VERSION=3.22.0

ENV CFLAGS="-std=gnu89 -Wno-implicit-int"

RUN echo "https://mirrors.aliyun.com/alpine/v3.22/main/" >> /etc/apk/repositories \
    && apk update \
    && apk add wget gcc g++ make bind-tools \
    && apk upgrade --no-cache \
    && wget https://ohse.de/uwe/releases/lrzsz-0.12.20.tar.gz \
    && tar -xf lrzsz-0.12.20.tar.gz \
    && cd lrzsz-0.12.20 \
    && ./configure \
    && make \
    && make install \
    && apk add --no-cache ca-certificates \
    && ln -s /usr/local/bin/lrz	/usr/bin/rz \
    && ln -s /usr/local/bin/lsz	/usr/bin/sz \
    && apk add bash-completion \
    && source /usr/share/bash-completion/bash_completion \
    && echo 'source <(kubectl completion bash)' >>~/.bashrc \
    && echo 'alias k=kubectl' >>~/.bashrc \
    && echo 'complete -o default -F __start_kubectl k' >>~/.bashrc

# https://github.com/flpanbin/ttyd/releases/download/1.7.8/ttyd.i686
RUN ttydArch="$(uname -m)" && echo "Building arch of $ttydArch.." \
    && curl -LO https://github.com/cloudtty/ttyd/releases/download/$TTYD_VERSION/ttyd.${ttydArch} \
    && chmod +x ttyd.${ttydArch} \
    && mv ttyd.${ttydArch} /usr/local/bin/ttyd \
    && which ttyd

# The base image's bundled helm binary is fixed to its build-time version, so
# apk upgrade never touches it. Replace it with a current release to pick up
# security fixes (e.g. CVE-2026-33186 in google.golang.org/grpc).
RUN helmArch="$(uname -m)" && case "${helmArch}" in \
        x86_64) helmArch=amd64 ;; \
        aarch64) helmArch=arm64 ;; \
    esac \
    && curl -LO https://get.helm.sh/helm-v${HELM_VERSION}-linux-${helmArch}.tar.gz \
    && tar -xzf helm-v${HELM_VERSION}-linux-${helmArch}.tar.gz \
    && mv linux-${helmArch}/helm /usr/local/bin/helm \
    && chmod +x /usr/local/bin/helm \
    && rm -rf helm-v${HELM_VERSION}-linux-${helmArch}.tar.gz linux-${helmArch} \
    && helm version

COPY --from=builder /app/dist/inline.html /usr/lib/ttyd/index.html
COPY ./docker/cloudshell/script/reset.sh  /usr/lib/ttyd/
COPY ./docker/cloudshell/script/startup.sh  /usr/lib/ttyd/

RUN mkdir /root/.kube \
    && chmod +x /usr/lib/ttyd/*.sh \
    && echo 'source /root/.env' >>~/.bashrc \
    && echo '' > .env

USER root
WORKDIR /root
