# Repository Instructions

## Project overview

This repository is a Go microservice backend for an IRCTC-style railway
booking system. User, booking, payment, and search services expose versioned
Protobuf APIs over native gRPC. PostgreSQL and Redis provide local persistence,
and Buf owns Protobuf linting and code generation.

Keep service boundaries explicit. Shared infrastructure belongs in `utils/`
only when it is genuinely reusable across services; service-specific business
logic belongs in that service's `internal` packages.

Browser-facing concerns do not belong in native gRPC services. A future
HTTP/GraphQL gateway owns cookies, CORS, CSRF protection, HTTP security
headers, and translation between browser requests and internal gRPC calls.

## Sources of truth

Before planning or implementing project work:

1. Read `features.md` for confirmed requirements and technology decisions.
2. Read root `guide.md` when it exists for system boundaries and request flows.
3. Before changing a service, read that service's `guide.md` when it exists.
4. Inspect the relevant code, configuration, tests, and generated contracts.

When a tutorial or transcript introduces new behavior, record only confirmed
requirements. Preserve existing decisions, avoid duplicates, and put unresolved
items under **Details not decided yet** instead of inventing behavior. Mark a
feature complete only after its implementation has passed relevant checks.

## Repository map

- `<service>/proto/`: source Protobuf contracts for that service.
- `<service>/server/`: service entry point and registration.
- `<service>/server/internal/`: service implementation details.
- `gen/go/`: generated Go Protobuf and gRPC bindings.
- `utils/`: shared configuration, clients, interceptors, errors, and mailer.
- `tests/`: explicitly tagged cross-component integration tests and fixtures.
- `docker/`: container initialization used by local infrastructure.
- `docker-compose.yml`: local PostgreSQL, Redis, and pgAdmin services.

## Local setup

From the repository root:

```bash
pnpm install
go mod download
cp .env.example .env.local
docker compose up -d
docker compose ps
```

Use the Go version declared in `go.mod`. Buf generation also requires
`protoc-gen-go` and `protoc-gen-go-grpc` on `PATH`. Keep local overrides and
real credentials in `.env.local` or `.env`, never in tracked files.

## Development workflow

- Work from the repository root and keep changes scoped to the requested task.
- Inspect the working tree before editing. Preserve unrelated user changes.
- Prefer the standard library and existing dependencies. Add a dependency only
  when it materially improves the implementation, and explain why.
- Implement the smallest complete slice; avoid speculative abstractions,
  premature service decomposition, and unrelated refactors.
- Update tests and relevant documentation when behavior or contracts change.
- Do not start long-running services unless needed for verification. Stop any
  process started solely for a test.
- Do not commit, push, amend, rebase, or rewrite history unless explicitly
  requested. When requested, use focused commits and verify the staged diff.

## Go conventions

- Run `gofmt` on every changed Go file.
- Write idiomatic, package-focused Go with clear names and small interfaces.
- Accept `context.Context` for request-scoped or I/O work and propagate
  cancellation and deadlines to database, Redis, mail, and gRPC operations.
- Wrap operational errors with useful context while preserving the cause with
  `%w`. At gRPC boundaries, return appropriate `status` and `codes` errors.
- Never call `log.Fatal`, `os.Exit`, or panic from handlers, interceptors,
  libraries, or cleanup paths. Only a process entry point may terminate the
  process for an unrecoverable startup failure.
- Close clients, pools, listeners, and other resources deterministically.
- Do not log passwords, tokens, authorization metadata, cookies, OTP values,
  full connection strings, or unnecessary personal data.
- Keep transport handling thin. Business rules should remain independently
  testable and should not depend directly on generated transport types when a
  domain boundary is warranted.

## Protobuf and gRPC

- Treat `.proto` files as the API source of truth. Never edit files under
  `gen/` manually.
- Preserve wire compatibility: do not reuse field numbers, silently change
  field meaning, or remove fields without an explicit compatibility decision.
- Keep package names and API versions aligned with their directory paths.
- Regenerate bindings whenever a Protobuf contract changes and review the
  generated diff for unintended churn.
- Value-embed the generated `Unimplemented<Service>Server` in service
  implementations for forward compatibility.
- Apply cross-cutting behavior through interceptors where appropriate. Return
  errors to gRPC; do not terminate the server from an interceptor.
- Keep health service registration and serving status accurate for every
  runnable service.

For Protobuf changes, run:

```bash
pnpm exec buf format --diff --exit-code
pnpm exec buf lint
pnpm exec buf build
pnpm exec buf generate
```

When changing an existing contract and a Git baseline is available, also run:

```bash
pnpm exec buf breaking --against '.git#branch=main'
```

## Data, configuration, and security

- Each service owns its PostgreSQL database. Integration tests use `tests_db`;
  they must never write to a service database.
- PostgreSQL initialization scripts run only when the data volume is first
  created. Do not assume editing a script migrates an existing volume.
- Use migrations or deliberate schema setup for application data. Do not rely
  on integration-test tables as application schema.
- Namespace Redis keys by service or feature, set a TTL when the data is
  temporary, and delete test keys during cleanup.
- Load configuration from environment variables. Keep real secrets out of
  source control and update `.env.example` with sanitized values when adding
  configuration.
- Validate configuration at the owning process boundary and return actionable
  startup errors without printing secret values.
- Use parameterized database operations and least-privilege database
  credentials. Do not share a service's database credentials with another
  service.

## Testing and verification

- Put unit tests beside the package they test using `*_test.go`.
- Keep external-infrastructure tests behind the `integration` build tag.
- Integration tests must use bounded contexts and `t.Cleanup`, create isolated
  data, verify observable behavior, and remove Redis keys, rows, and temporary
  tables even when an assertion fails.
- Do not make `go test ./...` depend on Docker or external services.
- Add regression coverage for bug fixes and test failure paths when practical.

Use the checks appropriate to the change:

```bash
# All normal Go packages
go test ./...
go vet ./...

# PostgreSQL and Redis integration flow; requires docker compose up -d
npm run tests:integration

# Compose syntax and resolved configuration
docker compose config
```

For service startup changes, run the affected `npm run start-<service>-service`
command and verify its gRPC health status. For broad shared changes, test every
affected service rather than only the service where the change originated.

## Documentation

- Keep architecture documentation factual and distinguish implemented behavior
  from planned work.
- Keep a `README.md` inside each service. List its RPCs first, including the
  implementation status and a short purpose for each RPC. Follow that with the
  service's feature list.
- Keep service feature descriptions at README depth. Mention the few technical
  choices needed to understand a feature, such as HMAC-backed OTP handling or
  Redis caching, without listing every request field or walking through the
  implementation line by line.
- Do not put environment-variable inventories, full payload descriptions,
  internal control flow, or step-by-step runbooks in a service README. Put that
  material in a guide or focused documentation when it is needed.
- Keep the root `README.md` brief. Give each service a top-level overview, list
  its exposed RPCs, and link every RPC to its section in the service README.
- When a feature is completed and verified, update the owning service README as
  part of the same feature set. Update the root summary when the service gains a
  new responsibility, adds an RPC, or changes public behavior.
- Before a requested feature commit, check that both documentation levels match
  the code. Do not document planned behavior as implemented.
- Write feature documentation in direct, everyday language. Remove promotional
  wording, filler, and unexplained jargon.
- Use a compact flow or ownership diagram only when it materially clarifies a
  multi-component interaction or state transition.
- Report verification commands and their results. If a required check could not
  run, state why; do not imply it passed.

## Code review rules

Prioritize findings that affect:

- correctness, data integrity, concurrency safety, or resource cleanup;
- Protobuf compatibility and gRPC status semantics;
- service ownership and accidental cross-database coupling;
- authentication, authorization, secret handling, or sensitive logging;
- timeouts, cancellation, retries, and failure behavior; and
- missing regression tests or verification for changed behavior.

Report concrete, actionable findings with file and line references. Do not
invent issues to fill a review, and do not hide a functional defect behind
style-only feedback.

## Definition of done

A change is complete when:

- the requested behavior is implemented without unrelated changes;
- affected code is formatted and relevant tests pass;
- Protobuf output is regenerated and validated when contracts change;
- configuration and documentation reflect the implemented behavior;
- generated changes correspond to their source contracts, and no secrets,
  temporary artifacts, or test data are left behind; and
- the final handoff summarizes the change, verification, and any remaining
  risks or follow-up work.
