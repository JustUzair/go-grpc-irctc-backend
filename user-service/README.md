# User service

The user service owns signup, user accounts, and authentication.

[Back to the project overview](../README.md)

## Supported RPCs

### `SendOTP`

Starts account signup and sends a six-digit verification code by email. The
caller receives a short-lived OTP session ID for the later verification step.

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
  Resend.
- Removes the pending OTP session when email delivery fails.

The current default is a five-minute OTP session and five code requests during
an active one-hour rate-limit window. The sixth request was manually verified
to return a rate-limit error.

## Planned RPCs and features

- OTP verification and user creation.
- User login, logout, and session management.
- Real user lookup through `GetUser`, which currently returns placeholder data.
- Gateway-owned OTP session cookies.
