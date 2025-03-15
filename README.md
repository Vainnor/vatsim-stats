# VATSIM Stats Collector

A service that collects and stores VATSIM network statistics, providing a REST API to access historical connection data and statistics.

## Configuration

The service requires the following environment variables:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=username
DB_PASSWORD=password
DB_NAME=dbname

# API Configuration
MASTER_API_KEY=your-secure-master-key    # Required for API key management
UPDATE_INTERVAL=15                        # Data update interval in minutes
```

## Rate Limiting and API Keys

The API implements rate limiting to ensure fair usage. By default, requests are limited to:
- 100 requests per 5-minute window per IP address
- Rate limit headers are included in responses:
  - `X-RateLimit-Limit`: Maximum requests per window
  - `X-RateLimit-Remaining`: Remaining requests in current window
  - `X-RateLimit-Reset`: Time when the current window resets (RFC3339 format)

### API Key Authentication

You can bypass rate limiting by using an API key. Include your API key in the `Authorization` header:

```http
Authorization: your-api-key-here
```

### API Key Management

API keys can be managed using the following endpoints. All management endpoints require the master API key for authentication.

#### Create API Key
```http
POST /api/keys
Authorization: master-key-here
Content-Type: application/json

{
    "description": "Key description here"
}
```

**Response:**
```json
{
    "id": 1,
    "key": "generated-api-key",
    "description": "Key description here",
    "created_at": "2024-03-15T12:00:00Z",
    "is_active": true
}
```

#### List API Keys
```http
GET /api/keys
Authorization: master-key-here
```

**Response:**
```json
[
    {
        "id": 1,
        "key": "api-key-1",
        "description": "Key description",
        "created_at": "2024-03-15T12:00:00Z",
        "last_used_at": "2024-03-15T13:00:00Z",
        "is_active": true
    }
]
```

#### Delete API Key
```http
DELETE /api/keys
Authorization: master-key-here
Content-Type: application/json

{
    "id": 1
}
```

## Available Endpoints

All endpoints (except API key management) are rate-limited and require the `/api` prefix.

### API Key Management (No Rate Limit)
- `/api/keys` - Create new API key (POST), List all API keys (GET), or Delete API key (DELETE)

### Data Endpoints (Rate Limited)
- `/api/membership/{cid}/{type}` - Get member connection history (type: pilot, atc, or atis)
- `/api/airports/{icao}/traffic` - Get current traffic information for a specific airport
- `/api/flights/search` - Search active flights with optional filters
- `/api/network/stats` - Get current network-wide statistics
- `/api/routes/popular` - Get most frequently flown routes
- `/api/routes/{origin}/{destination}/stats` - Get statistics for a specific route

### Analytics Endpoints (Rate Limited)
- `/api/analytics/network-stats` - Get network-wide statistics
- `/api/analytics/trends` - Get network trends (daily, weekly, monthly)

### Debug & Status (Rate Limited)
- `/api/membership/{cid}/debug` - Get debug information for a specific pilot
- `/api/collector/stats` - Get collector statistics

## API Documentation

### Membership Endpoints

#### Get Member Connection History
```http
GET /api/membership/{cid}/{type}
```

Returns historical connection data and statistics for a specific VATSIM member.

| Parameter | Type | Description |
|-----------|------|-------------|
| `cid` | string | VATSIM CID |
| `type` | string | Connection type: `pilot`, `atc`, or `atis` |

##### Pilot Connections
```http
GET /api/membership/{cid}/pilot
```

**Response:**
```json
{
  "items": [
    {
      "connection_id": {
        "id": 123,
        "vatsim_id": "1234567",
        "type": 1,
        "rating": 3,
        "callsign": "AAL123",
        "start": "2024-03-15T10:00:00Z",
        "end": "2024-03-15T12:30:00Z",
        "server": "USA-EAST"
      },
      "total_hours": 150,
      "total_flights": 42,
      "student_hours": 20,
      "ppl_hours": 40,
      "instrument_hours": 30,
      "cpl_hours": 40,
      "atpl_hours": 20,
      "current_session": {
        "start_time": "2024-03-15T10:00:00Z",
        "duration_minutes": 150,
        "has_flight_plan": true,
        "rating": 3
      }
    }
  ]
}
```

##### ATC Connections
```http
GET /api/membership/{cid}/atc
```

**Response:**
```json
{
  "items": [
    {
      "connection_id": {
        "id": 456,
        "vatsim_id": "1234567",
        "type": 2,
        "rating": 4,
        "callsign": "KJFK_TWR",
        "start": "2024-03-15T10:00:00Z",
        "end": "2024-03-15T12:30:00Z",
        "server": "USA-EAST"
      },
      "aircrafttracked": 25,
      "aircraftseen": 40,
      "flightsamended": 10,
      "handoffsinitiated": 15,
      "handoffsreceived": 12,
      "handoffsrefused": 2,
      "squawksassigned": 20,
      "cruisealtsmodified": 5,
      "tempaltsmodified": 8,
      "scratchpadmods": 30
    }
  ]
}
```

##### ATIS Connections
```http
GET /api/membership/{cid}/atis
```

**Response:**
```json
{
  "items": [
    {
      "connection_id": {
        "id": 789,
        "vatsim_id": "1234567",
        "type": 3,
        "rating": 4,
        "callsign": "KJFK_ATIS",
        "start": "2024-03-15T10:00:00Z",
        "end": "2024-03-15T12:30:00Z",
        "server": "USA-EAST"
      },
      "updates": 12,
      "frequency": "128.725",
      "letter": "A"
    }
  ]
}
```

### Airport Traffic Endpoint

#### Get Airport Traffic
```http
GET /api/airports/{icao}/traffic
```

Returns current traffic information for a specific airport, including active controllers, ATIS information, and flight movements.

| Parameter | Type | Description |
|-----------|------|-------------|
| `icao` | string | ICAO airport code |

**Response:**
```json
{
  "icao": "KJFK",
  "timestamp": "2024-03-15T12:00:00Z",
  "active_controllers": [
    {
      "position": "KJFK_TWR",
      "frequency": "118.700",
      "controller": {
        "cid": "1234567",
        "name": "John Doe",
        "rating": 4
      }
    }
  ],
  "atis": {
    "frequency": "128.725",
    "controller": "9876543"
  },
  "traffic": {
    "arrivals": [
      {
        "callsign": "DAL401",
        "aircraft": "B738",
        "altitude": 3000,
        "groundspeed": 160,
        "origin": "KBOS",
        "destination": "KJFK",
        "time": "2024-03-15T11:45:00Z"
      }
    ],
    "departures": [
      {
        "callsign": "UAL1234",
        "aircraft": "B77W",
        "altitude": 5000,
        "groundspeed": 250,
        "origin": "KJFK",
        "destination": "KLAX",
        "time": "2024-03-15T11:50:00Z"
      }
    ]
  },
  "statistics": {
    "hourly_movements": 45,
    "arrival_count": 20,
    "departure_count": 25
  }
}
```

### Flight Search Endpoint

#### Search Active Flights
```http
GET /api/flights/search
```

Search for active flights based on various criteria.

**Parameters:**
- `callsign` (query) - Filter by callsign (partial match)
- `aircraft` (query) - Filter by aircraft type
- `origin` (query) - Filter by departure airport
- `destination` (query) - Filter by arrival airport

**Response:**
```json
{
  "total": 25,
  "flights": [
    {
      "callsign": "BAW282",
      "aircraft": "B788",
      "origin": "EGLL",
      "destination": "KJFK",
      "altitude": 36000,
      "groundspeed": 480,
      "position": {
        "latitude": 51.4775,
        "longitude": -0.4614,
        "heading": 270
      },
      "flight_plan": "EGLL DCT KJFK",
      "start_time": "2024-03-15T10:30:00Z"
    }
  ]
}
```

### Network Statistics Endpoint

#### Get Network Statistics
```http
GET /api/network/stats
```

Returns current network-wide statistics.

**Response:**
```json
{
  "timestamp": "2024-03-15T12:00:00Z",
  "global": {
    "total_pilots": 1500,
    "total_atcs": 250,
    "active_pilots": 850
  },
  "server_stats": [
    {
      "name": "AMERICAS",
      "connected_users": 450
    }
  ],
  "rating_stats": [
    {
      "rating": 1,
      "pilot_count": 500,
      "atc_count": 100
    }
  ],
  "aircraft_stats": [
    {
      "type": "B738",
      "count": 125
    }
  ]
}
```

### Debug Endpoint

#### Get Pilot Debug Information
```http
GET /api/membership/{cid}/debug
```

Returns debug information for a specific pilot.

| Parameter | Type | Description |
|-----------|------|-------------|
| `cid` | string | VATSIM CID |

**Response:**
```json
{
  "cid": "1234567",
  "has_connection": true,
  "has_stats": true,
  "last_seen": "2024-03-15T11:55:00Z",
  "error": null
}
```

### Collector Status Endpoint

#### Get Collector Statistics
```http
GET /api/collector/stats
```

Returns current collector statistics.

**Response:**
```json
{
  "last_update": "2024-03-15T12:00:00Z",
  "total_snapshots": 1440,
  "active_pilots": 850,
  "processed_pilots": 15000,
  "start_time": "2024-03-14T00:00:00Z"
}
```

### Route Statistics Endpoints

#### Get Popular Routes
```http
GET /api/routes/popular
```

Returns the most frequently flown routes in the last 24 hours.

**Parameters:**
- `limit` (query, optional) - Number of routes to return (default: 10)

**Response:**
```json
{
  "timestamp": "2024-03-15T12:00:00Z",
  "routes": [
    {
      "origin": "KLAX",
      "destination": "KSFO",
      "flight_count": 42,
      "aircraft_types": [
        {
          "type": "B738",
          "count": 20
        },
        {
          "type": "A320",
          "count": 15
        }
      ],
      "avg_flight_time": 45,
      "avg_altitude": 32000,
      "avg_groundspeed": 450,
      "active_flights": 5
    }
  ],
  "total": 1
}
```

#### Get Route Statistics
```http
GET /api/routes/{origin}/{destination}/stats
```

Returns detailed statistics for a specific route.

| Parameter | Type | Description |
|-----------|------|-------------|
| `origin` | string | ICAO code of departure airport |
| `destination` | string | ICAO code of arrival airport |

**Response:**
```json
{
  "origin": "KLAX",
  "destination": "KSFO",
  "flight_count": 42,
  "aircraft_types": [
    {
      "type": "B738",
      "count": 20
    },
    {
      "type": "A320",
      "count": 15
    }
  ],
  "avg_flight_time": 45,
  "avg_altitude": 32000,
  "avg_groundspeed": 450,
  "active_flights": 5
}
```

## Response Types

### Connection Types
- `1`: Pilot
- `2`: ATC
- `3`: ATIS

### Rating Values
- `1`: Student
- `2`: PPL (Private Pilot)
- `3`: Instrument
- `4`: CPL (Commercial Pilot)
- `5`: ATPL (Airline Transport Pilot)

## Error Responses

All endpoints may return the following error responses:

```json
{
  "error": "Error message describing what went wrong"
}
```

Common HTTP status codes:
- 200: Success
- 400: Bad Request
- 404: Not Found
- 429: Too Many Requests
- 500: Internal Server Error

## Features

- Fetches VATSIM network data every 15 seconds (configurable)
- Stores data only when changes are detected
- Maintains historical data of pilots, flight plans, and network statistics
- Uses efficient database transactions for data integrity

## Prerequisites

- Go 1.22 or later
- PostgreSQL 12 or later
- Git

## Installation

1. Clone the repository:
```bash
git clone https://github.com/vainnor/vatsim-stats.git
cd vatsim-stats
```

2. Install dependencies:
```bash
go mod download
```

3. Copy the example environment file and configure it:
```bash
cp .env.example .env
```

Edit the `.env` file with your PostgreSQL credentials and desired settings:
```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=vatsim_stats
DB_USER=your_username
DB_PASSWORD=your_password
UPDATE_INTERVAL=15
```

## Usage

1. Create the PostgreSQL database:
```bash
createdb vatsim_stats
```

2. Run the application:
```bash
go run main.go
```

## Database Schema

The application uses several tables to store and manage VATSIM network data:

### Core Tables
- `snapshots`: Stores general network information for each data update
- `facilities`: Stores facility information (e.g., FSS, DEL, GND, TWR)
- `ratings`: Stores controller rating information
- `pilot_ratings`: Stores pilot rating information
- `military_ratings`: Stores military rating information
- `pilots`: Stores pilot information linked to snapshots
- `controllers`: Stores controller information linked to snapshots
- `flight_plans`: Stores flight plan information linked to pilots
- `connections`: Stores historical connection data for pilots and controllers
- `api_keys`: Stores API keys for rate limit bypassing

### Statistics Tables
- `atc_stats`: Stores controller statistics (aircraft tracked, handoffs, etc.)
- `pilot_stats`: Stores pilot statistics (flight time, rating, etc.)
- `pilot_total_stats`: Stores aggregated pilot statistics
- `atis_stats`: Stores ATIS connection statistics
- `airport_stats`: Stores airport movement statistics
- `network_stats`: Stores network-wide statistics
- `server_stats`: Stores per-server statistics
- `rating_stats`: Stores statistics by rating
- `aircraft_stats`: Stores statistics by aircraft type
- `route_stats`: Stores route usage statistics

### Trend Tables
- `network_trends_daily`: Stores daily network statistics
- `network_trends_weekly`: Stores weekly network statistics
- `network_trends_monthly`: Stores monthly network statistics

Each table includes appropriate indexes and foreign key relationships to maintain data integrity and query performance.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details. 