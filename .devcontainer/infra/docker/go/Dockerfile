FROM golang:1.24-bullseye

ARG USER_NAME
ARG GROUP_NAME
ARG UID
ARG GID
ARG WORKDIR
ARG TZ

ENV TZ=${TZ}
ENV HOME=/home/${USER_NAME}

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      vim \
      make && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd -g ${GID} ${GROUP_NAME} && \
    useradd -lm -s /bin/bash -u ${UID} -g ${GID} ${USER_NAME}

USER ${USER_NAME}

RUN mkdir -p ${WORKDIR}
WORKDIR ${WORKDIR}

ENV GOPATH=/home/${USER_NAME}/go
ENV PATH=${PATH}:${GOPATH}/bin

CMD ["bash"]
