# Logging setup

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
