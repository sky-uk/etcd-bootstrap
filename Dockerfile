FROM golang:1.9

ENV DEBIAN_FRONTEND=noninteractive
RUN  apt-get update \
  && apt-get install -y software-properties-common python-pip \
  python-setuptools \
  python-dev \
  build-essential \
  libssl-dev \
  libffi-dev \
  && apt-get install --no-install-suggests --no-install-recommends -y \
  curl \
  git \
  build-essential \
  python-netaddr \
  unzip \
  vim \
  wget \
  inotify-tools \
  && apt-get clean -y \
  && apt-get autoremove -y \
  && rm -rf /var/lib/apt/lists/* /tmp/*

RUN pip install pyinotify

# Grab the source code and add it to the workspace.
ENV PATHWORK=/go/src/github.com/sky-uk/etcd-bootstrap
ADD ./ $PATHWORK
WORKDIR $PATHWORK

RUN go get -u github.com/kardianos/govendor
RUN go get -u golang.org/x/tools/cmd/goimports
RUN go get -u golang.org/x/lint/golint 

ADD ./docker/* /
RUN chmod 755 /entrypoint.sh
RUN chmod 755 /autocompile.py
CMD /entrypoint.sh
