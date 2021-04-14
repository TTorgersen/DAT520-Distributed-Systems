# Start from a image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

# Copy dependencies from your lab3 files to the container's workspace.
COPY ./lab3 /go/src/app/lab3

# Copy the local package files (lab4) to the container's workspace.
COPY . /go/src/app
COPY ./lab4/app/netconf.json /go/src/app/netconf.json

WORKDIR /go/src/app

#Build and install your application inside the container.
RUN go install -v ./lab4/client

ENTRYPOINT ["/go/bin/client"]