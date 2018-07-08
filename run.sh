#!/bin/bash

if [ "$APP_ENV" = "production" ]; \
    then \
    ./exchange; \
    else \
    go get && \
    go get github.com/cespare/reflex && \
    reflex -r '\.go|json$' -s -- sh -c 'go build -o exchange && SIMULATION=$SIMULATION DEBUG=true ./exchange'; \
fi
