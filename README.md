# Go-IRCTC (WIP)

Go-IRCTC rebuilds the backend domain of India's IRCTC railway booking
experience with Go and native gRPC microservices. Each service exposes a
versioned Protobuf API and owns the data for its part of the system. PostgreSQL
provides service-owned databases, while Redis handles short-lived state such as
OTP signup sessions and request limits.

Core stack: Go 1.26, gRPC, Protobuf, Buf, PostgreSQL, GORM, Redis, Docker
Compose, and Resend.

## Services

### User service

Handles signup, user accounts, and authentication. The current implementation
supports email verification signup and password login with signed access and
refresh tokens.

Supported RPCs:

- [`GetUser`](user-service/README.md#getuser) — contract present; lookup behavior
  is not implemented yet.
- [`SendOTP`](user-service/README.md#sendotp) — implemented; starts signup and
  sends the email verification code.
- [`VerifyOTP`](user-service/README.md#verifyotp) — implemented; consumes the
  OTP and creates the verified user.
- [`Login`](user-service/README.md#login) — implemented; validates credentials
  and returns access/refresh tokens to the gateway.
- [`RotateRefreshToken`](user-service/README.md#rotaterefreshtoken) — implemented;
  rotates the refresh session and issues a new token pair.

[Read the user service feature summary](user-service/README.md#features)

The booking, payment, and search services are set up but do not have completed
business features yet. Their RPC links and summaries will be added as those
features are built and tested.

## Logging setup

Copy the example environment file before starting a service:

```bash
cp .env.example .env.local
```

The default configuration emits compact, colorized logs at `INFO` level:

```dotenv
LOG_LEVEL=info
LOG_FORMAT=text
```

`info` includes `INFO`, `WARN`, and `ERROR` entries. To include additional debugging logs, change the level:

```dotenv
LOG_LEVEL=debug
```

To keep compact text logs without ANSI colors:

```dotenv
LOG_FORMAT=text
NO_COLOR=1
```

For structured production logs, use JSON:

```dotenv
LOG_LEVEL=info
LOG_FORMAT=json
```

Example development output:

```text
10:30:13.308 INF finished call rpc.service=user.v1.UserService rpc.method=GetUser rpc.code=OK duration=23.041µs
```
