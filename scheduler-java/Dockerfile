FROM gradle:jdk14

ARG port=10600

COPY . /opt/scheduler-java
WORKDIR /opt/scheduler-java

EXPOSE {port}

ENTRYPOINT [ 'gradle', 'run' ]
