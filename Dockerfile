FROM golang:1.23-bullseye AS build-env

WORKDIR /go/src/github.com/guru/guru

RUN apt-get update -y
RUN apt-get install git -y

COPY . .

RUN make build

FROM golang:1.23-bullseye

RUN apt-get update -y
RUN apt-get install ca-certificates jq -y

WORKDIR /root

COPY --from=build-env /go/src/github.com/guru/guru/build/gurud /usr/bin/gurud

EXPOSE 26656 26657 1317 9090 8545 8546

CMD ["gurud"]
