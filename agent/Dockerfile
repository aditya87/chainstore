FROM golang:1.9.2

RUN mkdir -p /src/github.com/aditya87/chainstore/agent
ENV GOPATH=/
ADD . /src/github.com/aditya87/chainstore/agent

WORKDIR /src/github.com/aditya87/chainstore/agent
RUN go build .
WORKDIR /src/github.com/aditya87/chainstore/agent_test
RUN go build .

FROM redis

RUN mkdir -p /app
COPY --from=0 /src/github.com/aditya87/chainstore/agent/agent /app/agent
COPY --from=0 /src/github.com/aditya87/chainstore/agent_test/agent_test /app/agent_test
CMD /app/agent_test
