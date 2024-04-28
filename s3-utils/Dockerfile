FROM docker.io/python:3.11

ARG TARGETARCH

ENV HOME=/root

RUN \
  apt update -y \
  && \
  apt install -y git-lfs zip \
  && \
  pip3 --no-cache-dir install boto3 requests torch-model-archiver \
  && \
  curl -Lo /usr/local/bin/mc https://dl.min.io/client/mc/release/linux-${TARGETARCH}/mc \
  && \
  chmod 775 /usr/local/bin/mc \
  && \
  curl -Lo /tmp/oc.tgz "https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.15.5/openshift-client-linux-${TARGETARCH}-rhel9.tar.gz" \
  && \
  mkdir /tmp/install \
  && \
  cd /tmp/install \
  && \
  tar -zxf /tmp/oc.tgz \
  && \
  rm /tmp/oc.tgz \
  && \
  mv oc kubectl /usr/local/bin/ \
  && \
  chmod g+rwx /usr/local/bin/oc /usr/local/bin/kubectl \
  && \
  chown :0 /usr/local/bin/oc /usr/local/bin/kubectl \
  && \
  rm -rf /tmp/install

WORKDIR /app
COPY --chown=:0 --chmod=775 s3_utils.py /app/s3_utils.py

RUN \
  chown -R :0 /root /app \
  && \
  chmod -R g=u /root /app
