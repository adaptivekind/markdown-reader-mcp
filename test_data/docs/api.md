# API Documentation

This document describes the REST API endpoints.

## Authentication

All API calls require authentication using Bearer tokens.

## Endpoints

### GET /users
Returns a list of users.

### POST /users
Creates a new user with the following fields:
- name (required)
- email (required)
- age (optional)

## Error Handling

The API returns standard HTTP status codes and JSON error responses.
