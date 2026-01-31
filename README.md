# go-idempotency

HTTP middleware for idempotent request handling in Go. Prevents duplicate processing of requests by caching responses based on idempotency keys—essential for payment, financial, and other critical APIs.

## Features

- 🔒 **Prevents duplicate operations** using idempotency keys
- 🚀 **Multiple storage backends**: in-memory and Redis
- ⚡ **Concurrent request handling** with distributed locking
- 🎯 **Request fingerprinting** (method + path + body hash)
- ⏱️ **Configurable TTL** for cached responses
- 🧪 **Thoroughly tested** with 90%+ coverage
- 📦 **Zero dependencies** (except storage drivers)

## Installation

```bash
go get github.com/AnandSundar/go-idempotency
