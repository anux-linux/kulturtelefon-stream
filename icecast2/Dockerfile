FROM debian:stable-slim

LABEL maintainer="Antonio Mark https://github.com/anux-linux"

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y icecast2 && \
    apt-get -y autoclean && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    chown -R icecast2 /etc/icecast2 



COPY icecast.xml /etc/icecast2/icecast.xml
COPY startup.sh /startup.sh

RUN chmod +x /startup.sh && \
    chmod 644 /etc/icecast2/icecast.xml && \
    chown icecast2:icecast2 /etc/icecast2/icecast.xml && \
    chown icecast2:icecast2 /startup.sh

CMD ["/startup.sh"]
EXPOSE 8000

VOLUME ["/config", "/var/log/icecast2", "/etc/icecast2", "/etc/icecast2/mounts"]