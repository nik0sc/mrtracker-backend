#!/bin/sh

if [ -z "$GOPATH" ]; then
    echo "where is \$GOPATH?"
    exit 1
fi

cd "`dirname "$0"`"

export PATH="$GOPATH/bin:$PATH"

if [ ! -x "`which protoc`" ]; then
    echo "where is executable protoc?"
    exit 1
fi

if [ ! -x "`which protoc-gen-go`" ]; then
    echo "where is executable protoc-gen-go?"
    exit 1
fi

protoc \
    --go_out=. \
    --go_opt=module=go.lepak.sg/mrtracker-backend/proto \
    *.proto