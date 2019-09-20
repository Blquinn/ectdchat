#!/bin/bash

protoc -I protobufs/ protobufs/chatserver.proto --go_out=plugins=grpc:protobufs

