# Notification service

The notification service consumes Kafka events and delivers email through the
configured mail provider. It is not a browser-facing API.

## Supported RPCs

There are no application RPCs yet. The service registers the standard gRPC
health service for process checks.

## Features

- Consumes OTP notification events from `notification.otp-email`.
- Consumes welcome events from `notification.welcome-email`.
- Decodes the shared JSON event types and renders the matching email template.
- Uses a dedicated Kafka consumer group so notification processing is owned by
  this service.
- Leaves booking and payment notification topics subscribed but unsupported
  until their handlers are implemented.
