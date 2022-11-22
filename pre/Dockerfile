FROM golang:1.18.3-stretch

RUN apt update
RUN mkdir pre
COPY . pre/
RUN cd pre; go get -u; chmod +x build.sh; ./build.sh
CMD ["./pre/main"]