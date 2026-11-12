FROM ubuntu:jammy

ARG DEBIAN_FRONTEND=noninteractive
ARG TARGETARCH
ARG TZ="Asia/Taipei"

RUN  apt-get -qq update && \
     apt-get -qq install -y --no-install-recommends tzdata ca-certificates libglib2.0-0 libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 libcups2 libdrm2 \
         libdbus-1-3 libexpat1 libxcb1 libxkbcommon0 libx11-6 libxcomposite1 libxdamage1 libxext6 libxfixes3 libxrandr2 libgbm1 libpango-1.0-0 \
         libcairo2 libasound2 libatspi2.0-0 && \
     ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime && \
     echo ${TZ} > /etc/timezone && \
     dpkg-reconfigure --frontend noninteractive tzdata &&\
     rm -rf /var/lib/apt/lists/* && \
     mkdir -p /opt/config/

COPY conf/config_default.yml /opt/config/config.yml
COPY conf/QuestionBank.db /opt/QuestionBank.db
# 二進位檔案由 workflow 預先複製到根目錄
COPY studyclaw /opt/studyclaw

# Playwright driver 1.42.1 and Chromium are preinstalled in the upstream image.
# The azureedge.net CDN used by playwright-go's runtime installer now returns
# 404, so installing them on startup fails and the study modules panic with a
# nil pointer. Copying them at build time avoids that download entirely.
# BuildKit resolves --from to the matching target platform (amd64 / arm64).
COPY --from=ghcr.io/legolasljl/studyclaw:latest /opt/tools /opt/tools

RUN  chmod -R 777 /opt/studyclaw && \
     cd /opt/ && \
     ./studyclaw --init

EXPOSE 8080

VOLUME /opt/config

CMD cd /opt && ./studyclaw
