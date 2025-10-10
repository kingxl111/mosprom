FROM alpine:3.13

RUN apk update && \
    apk upgrade && \
    apk add bash && \
    apk add curl &&  \
    apk add tar &&    \
    rm -rf /var/cache/apk/*


RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-amd64.tar.gz | tar xvz

RUN chmod +x migrate && \
    mv migrate /bin/migrate

WORKDIR /app

COPY ./migrations/*.sql migrations/
COPY migrator.sh .
COPY .env .

RUN chmod +x migrator.sh

ENTRYPOINT ["bash", "migrator.sh"]