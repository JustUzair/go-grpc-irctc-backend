# User service

The user service owns signup, user accounts, and authentication.

[Back to the project overview](../README.md)

## Supported RPCs

### `GetUser`

Contract present; the lookup behavior is not implemented yet.

### `SendOTP`

Starts account signup and sends a six-digit verification code by email. The
caller receives a short-lived OTP session ID for the later verification step.

### `VerifyOTP`

Consumes the OTP session, creates the verified user account, and sends a
confirmation email.

### `Login`

Checks credentials and returns signed access and refresh tokens with the
authenticated user's public profile and token lifetimes.

### `RotateRefreshToken`

Validates the current refresh session, rejects a reused token, and returns a
new access-token and refresh-token pair.

### `VerifyGoogleIDToken`

Validates a Google ID token, links it to an existing local account with the
same verified email, or creates a new local account. It returns service-issued
access and refresh tokens.

## Features

### Email OTP signup

- Checks PostgreSQL before starting signup for an email that already belongs to
  a user.
- Hashes pending passwords with bcrypt.
- Generates OTPs with `crypto/rand` and stores an HMAC-backed check instead of
  the plain code.
- Caches pending signup sessions in Redis with a short expiry.
- Limits repeated OTP requests per email with an expiring Redis counter.
- Renders the signup email from an embedded HTML template and sends it through
  the configured mail provider.
- Removes the pending OTP session when email delivery fails.
- Issues signed access and refresh tokens after a successful password check.
- Stores refresh-token JTIs in Redis and rotates them after a successful
  refresh.
- Supports Google Identity Services sign-in through validated ID tokens.
- Links external identities to local accounts through a PostgreSQL
  auth-provider model with composite uniqueness constraints.
- Captures request metadata through a gRPC interceptor for future session-risk
  handling.

The current default is a five-minute OTP session and five code requests during
an active one-hour rate-limit window. The sixth request was manually verified
to return a rate-limit error.

## Planned RPCs and features

- Logout and server-side session revocation.
- Gateway-owned OTP session cookies.
