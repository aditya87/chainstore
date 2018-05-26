FROM golang:1.9.2

ENV GOPATH=/
ENV PATH=$PATH:/bin

RUN mkdir -p /src/github.com/aditya87/chainstore
ADD . /src/github.com/aditya87/chainstore

WORKDIR /src/github.com/aditya87/chainstore/agent
RUN go build .
WORKDIR /src/github.com/aditya87/chainstore/integration
RUN go build .
WORKDIR /src/github.com/aditya87/chainstore/startup
RUN go build .
WORKDIR /src/github.com/aditya87/chainstore/startup/agent
RUN go build -o agent-start .
WORKDIR /src/github.com/aditya87/chainstore/startup/redis
RUN go build -o redis-start .

FROM redis

RUN mkdir -p /app
COPY --from=0 /src/github.com/aditya87/chainstore/agent/agent /app/agent
COPY --from=0 /src/github.com/aditya87/chainstore/startup/agent/agent-start /app/agent-start
COPY --from=0 /src/github.com/aditya87/chainstore/startup/redis/redis-start /app/redis-start
COPY --from=0 /src/github.com/aditya87/chainstore/startup/startup /app/startup

CMD /app/startup && /bin/bash
