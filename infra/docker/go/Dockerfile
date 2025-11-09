FROM golang:1.24-bullseye

ARG USER_NAME
ARG GROUP_NAME
ARG UID
ARG GID
ARG WORKDIR
ARG TZ

ENV TZ=${TZ}

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      curl && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd -g ${GID} ${GROUP_NAME} && \
    useradd -lm -s /bin/bash -u ${UID} -g ${GID} ${USER_NAME}

USER ${USER_NAME}

ENV HOME=/home/${USER_NAME}
ENV GOPATH=${HOME}/go
ENV PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin

RUN mkdir -p ${WORKDIR}
WORKDIR ${WORKDIR}

CMD ["bash"]
