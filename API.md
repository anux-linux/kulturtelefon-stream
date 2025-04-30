# Streaming API Documentation

This document describes the endpoints and functionality of the Streaming API service, which provides management of Icecast stream mount points.

## Base URL

All endpoints are relative to the base URL of the API server.

## Authentication

Most endpoints require authentication using a token in the `Authorization` header.

### Getting a Token

**Endpoint**: `GET /user/token`

**Authentication**: None (public endpoint)

**Query Parameters**:
- `username`: The username for authentication
- `password`: The password for authentication

**Response**:
```json
{
  "token": "your-authentication-token"
}
```

**Status Codes**:
- `200 OK`: Successful authentication
- `400 Bad Request`: Invalid credentials

## Public Endpoints

These endpoints do not require authentication.

### Health Check

**Endpoint**: `GET /public/health`

**Authentication**: None

**Response**:
```json
{
  "status": "ok"
}
```

**Status Codes**:
- `200 OK`: Service is healthy

### Version Information

**Endpoint**: `GET /public/version`

**Authentication**: None

**Response**:
```json
{
  "version": "0.0.1"
}
```

**Status Codes**:
- `200 OK`: Version information retrieved successfully

## Stream Management Endpoints

These endpoints require authentication with a valid token.

### Create Stream

**Endpoint**: `POST /api/streams`

**Authentication**: Required (token with `post_stream` permission)

**Request Body**:
```json
{
  "mount-name": "/stream-name.mp3",
  "username": "streamuser",
  "password": "streampassword",
  "public": 1,
  "stream-name": "My Stream",
  "stream-description": "A description of my stream"
}
```

**Response**: The created mount point configuration

**Status Codes**:
- `201 Created`: Stream created successfully
- `400 Bad Request`: Invalid JSON or other error
- `401 Unauthorized`: Missing or invalid authentication

### List All Streams

**Endpoint**: `GET /api/streams`

**Authentication**: Required (token with `get_all_streams` permission)

**Response**: Array of all available mount point configurations

**Status Codes**:
- `200 OK`: Streams retrieved successfully
- `400 Bad Request`: Database error
- `401 Unauthorized`: Missing or invalid authentication

### Get Stream Details

**Endpoint**: `GET /api/streams/{streamName}`

**Authentication**: Required (token with `get_stream` permission)

**URL Parameters**:
- `streamName`: The name of the stream to retrieve

**Response**: The requested mount point configuration

**Status Codes**:
- `200 OK`: Stream retrieved successfully
- `400 Bad Request`: Missing stream name or database error
- `401 Unauthorized`: Missing or invalid authentication

### Update Stream

**Endpoint**: `POST /api/streams/{streamName}`

**Authentication**: Required (token with `post_stream` permission)

**URL Parameters**:
- `streamName`: The name of the stream to update

**Request Body**:
```json
{
  "username": "newstreamuser",
  "password": "newstreampassword",
  "public": 1,
  "stream-name": "Updated Stream Name",
  "stream-description": "Updated stream description"
}
```

**Response**: The updated mount point configuration

**Status Codes**:
- `200 OK`: Stream updated successfully
- `400 Bad Request`: Invalid JSON, missing stream name, or database error
- `401 Unauthorized`: Missing or invalid authentication

### Delete Stream

**Endpoint**: `DELETE /api/streams/{streamName}`

**Authentication**: Required (token with `delete_stream` permission)

**URL Parameters**:
- `streamName`: The name of the stream to delete

**Response**:
```json
{
  "status": "deleted"
}
```

**Status Codes**:
- `200 OK`: Stream deleted successfully
- `400 Bad Request`: Missing stream name or database error
- `401 Unauthorized`: Missing or invalid authentication

## Error Responses

All API errors are returned in the following format:

```json
{
  "error": "Error message describing the issue"
}
```

## Permissions

The API uses the following permission types:
- `post_stream`: Create or update streams
- `get_all_streams`: List all streams
- `get_stream`: Get details of a specific stream
- `delete_stream`: Delete a stream

## Technical Notes

- The API stores stream configurations in both a database and Icecast configuration files
- Authentication tokens expire after 1 year from creation
- The API uses Go's standard HTTP libraries with custom middleware for authentication and logging
