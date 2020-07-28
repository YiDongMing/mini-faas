#! /bin/bash
FROM debian:stable-slim

ARG port=10600
ARG server_dir=/aliyuncnpc/scheduler
ARG config_dir=${server_dir}/config
ARG log_dir=${server_dir}/log

ENV SERVICE_PORT=${port}

RUN mkdir -p ${server_dir}
RUN mkdir -p ${config_dir}

COPY ./ ${server_dir}/
COPY ./config/dev/config.json ${config_dir}/
COPY ./config/dev/log.xml ${config_dir}/

WORKDIR ${server_dir}
EXPOSE ${port}

ENTRYPOINT ["/aliyuncnpc/scheduler/scheduler", \
            "-config=./config/config.json", \
            "-logconfig=./config/log.xml"]
