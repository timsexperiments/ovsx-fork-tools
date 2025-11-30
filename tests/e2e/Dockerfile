FROM golang:1.24

RUN apt-get update && apt-get install -y \
    curl \
    git \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

ARG ACT_VERSION=0.2.82
ARG ACT_CHECKSUM=76645c0bbe4bb69a8f0ba3caefa681b2f4c04babd4679c67861af9a276a3561f
RUN curl -L -o act.tar.gz https://github.com/nektos/act/releases/download/v${ACT_VERSION}/act_Linux_x86_64.tar.gz && \
    echo "${ACT_CHECKSUM}  act.tar.gz" | sha256sum -c - && \
    tar -xzf act.tar.gz -C /usr/local/bin act && \
    rm act.tar.gz

WORKDIR /app

CMD ["go", "test", "-v", "./tests/e2e/..."]
