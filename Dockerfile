FROM alpine
ENV ORG_PATH="github.com/x140cc/shorturl-docker"
ENV REPO_PATH="${ORG_PATH}"
ADD entry_point.sh /entry_point.sh
RUN apk --update add go git \
  && export GOPATH=/gopath \
  && go get ${ORG_PATH} \
  && mkdir -p $GOPATH/src/${ORG_PATH} \
  && cd $GOPATH/src/${ORG_PATH} \
  && CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-s" -o /shorturl ${REPO_PATH} \
  && apk del go git \
  && rm -rf $GOPATH /var/cache/apk/* \
  && mkdir -p /db
VOLUME /db
EXPOSE 8080
ENTRYPOINT ["/bin/sh", "/entry_point.sh"]
