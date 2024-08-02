#!/bin/sh

protoc --proto_path=. --go_out=. --go_opt=paths=source_relative messages.proto messages-common.proto messages-ethereum.proto messages-management.proto messages-crypto.proto
