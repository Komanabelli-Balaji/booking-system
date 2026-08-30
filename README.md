# Cinema Booking System

A real-time cinema seat booking system built with **Go** and **Redis**. Users can browse movies, hold seats with a time-limited reservation, and confirm or release bookings, all with live updates across multiple browser sessions.

## Features

- **Seat Holds with TTL**: Seats are temporarily held for 2 minutes using Redis key expiration. If the user doesn't confirm in time, the seat is automatically released.
- **Atomic Seat Locking**: Uses Redis `SET NX` to guarantee that only one user can hold a given seat, even under heavy concurrency (tested with 100k goroutines).
- **Real-Time Seat Map**: Uses WebSockets and Redis Keyspace Notifications to push seat availability updates instantly to all connected clients viewing the same movie.
- **Session Ownership Validation**: Confirm and release operations verify that the requesting user owns the session before proceeding.
- **Interactive Frontend**: A vanilla HTML/CSS/JS interface with a cinema-style seat grid, countdown timer, and checkout panel.

## Tech Stack

| Layer     | Technology                |
|-----------|---------------------------|
| Backend   | Go 1.26 (stdlib `net/http`) |
| Storage   | Redis 7                   |
| Frontend  | Vanilla HTML / CSS / JS   |
| Tooling   | Docker, Docker Compose    |

## Project Structure

```
booking-system/
├── cmd/
│   └── main.go                         # Entrypoint — HTTP server, routes, movie data
├── internal/
│   ├── adapters/
│   │   └── redis/
│   │       └── redis.go                # Redis client factory with health check
│   ├── booking/
│   │   ├── domain.go                   # Booking model, BookingStore interface, errors
│   │   ├── service.go                  # Business logic layer (delegates to store)
│   │   ├── handler.go                  # HTTP handlers (hold, list, confirm, release)
│   │   ├── redis_store.go              # Redis-backed BookingStore implementation
│   │   ├── memory_store.go             # Simple in-memory store (no concurrency safety)
│   │   ├── concurrent_store.go         # Mutex-protected in-memory store
│   │   └── service_test.go             # Concurrency test (100k goroutines race for 1 seat)
│   ├── utils/
│   │   └── utils.go                    # JSON response helper
│   └── ws/
│       ├── hub.go                      # WebSocket Hub managing clients per movie
│       ├── client.go                   # WebSocket connection wrapper
│       └── hub_test.go                 # Hub unit tests
├── static/
│   ├── index.html                      # Single-page app shell
│   ├── style.css                       # Cinema-themed UI styles
│   └── app.js                          # Client-side logic (seat grid, polling, checkout)
├── docker-compose.yaml                 # App, Redis + Redis Commander services
├── Dockerfile                          # Multi-stage Go application build
├── go.mod
└── go.sum
```

## Architecture

```
┌────────────┐          ┌──────────────┐        ┌─────────────────┐         ┌───────────┐
│  Browser   │──HTTP──▶│  net/http    │──────▶ │  booking.Service│──────▶ │   Redis   │
│  (static/) │◀─JSON───│  (handlers)  │◀────── │ (business logic)│◀────── │  (store)  │
└─────┬──────┘          └──────────────┘        └─────────────────┘         └─────┬─────┘
      │                          ▲                                                │
      │   WebSocket              │               (Broadcaster)                    │
      └──────────────────────────┴─────────────── ws.Hub ◀───(Keyspace Expiry)────┘
```

The codebase follows a **layered architecture**:

1. **Handler**: Parses HTTP requests, calls the service, writes JSON responses.
2. **Service**: Thin orchestration layer over `BookingStore` interface.
3. **Store**: Swappable storage backends implementing `BookingStore`:
   - `RedisStore`: Production store with atomic operations and TTL-based expiration.
   - `ConcurrentStore`: In-memory store guarded by `sync.RWMutex`.
   - `MemoryStore`: Simplest implementation (no concurrency safety, useful for prototyping).

## API Reference

### List Movies

```
GET /movies
```

Returns the catalog of available movies with their seating layout.

**Response** `200 OK`
```json
[
  { "id": "paradise", "title": "Paradise", "rows": 6, "seats_per_row": 10 },
  { "id": "bnd", "title": "Spider-Man: Brand New Day", "rows": 7, "seats_per_row": 10 },
  { "id": "kgf", "title": "KGF", "rows": 8, "seats_per_row": 10 }
]
```

---

### List Seats

```
GET /movies/{movieID}/seats
```

Returns all currently held or confirmed seats for a movie.

**Response** `200 OK`
```json
[
  { "seat_id": "A1", "user_id": "abc123", "booked": true, "confirmed": false },
  { "seat_id": "B5", "user_id": "def456", "booked": true, "confirmed": true }
]
```

---

### Hold a Seat

```
POST /movies/{movieID}/seats/{seatID}/hold
```

Temporarily reserves a seat for 2 minutes.

**Request Body**
```json
{ "user_id": "abc123" }
```

**Response** `201 Created`
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "movieID": "paradise",
  "seat_id": "A1",
  "expires_at": "2026-08-17T19:05:00Z"
}
```

---

### Confirm a Session

```
PUT /sessions/{sessionID}/confirm
```

Permanently books the held seat (removes TTL).

**Request Body**
```json
{ "user_id": "abc123" }
```

**Response** `200 OK`
```json
{
  "session_id": "550e8400-...",
  "movie_id": "paradise",
  "seat_id": "A1",
  "user_id": "abc123",
  "status": "confirmed"
}
```

---

### Release a Session

```
DELETE /sessions/{sessionID}
```

Manually releases a held seat before it expires.

**Request Body**
```json
{ "user_id": "abc123" }
```

**Response** `204 No Content`

## Getting Started

### Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/)

### 1. Run Everything via Docker

You can easily build the Go application and spin it up along with Redis using Docker Compose:

```bash
docker compose up -d --build
```

This starts:
- **booking-app** (Go Server) on `localhost:8080`
- **Redis** on `localhost:6379`
- **Redis Commander** (web UI) on `localhost:8081`

### 2. Local Development (Alternative)

If you prefer to run the Go application natively while keeping Redis in Docker:

```bash
docker compose up -d redis redis-commander
go run ./cmd
```

The server starts on [http://localhost:8080](http://localhost:8080).

### 3. Open the App

Navigate to [http://localhost:8080](http://localhost:8080) in your browser. Open multiple tabs to see real-time seat updates across sessions.

## How It Works

1. **Browse**: The landing page loads the movie catalog from `GET /movies`.
2. **Select a Movie**: Click a movie card to view its seat grid. The frontend connects to the WebSocket endpoint `/ws` and subscribes to that movie's updates.
3. **Hold a Seat**: Click an available seat. The server atomically sets a Redis key with `SET NX` and a 2-minute TTL, and triggers a WebSocket broadcast. A checkout panel appears with a countdown timer.
4. **Confirm or Release**: Confirm removes the TTL (seat is permanently booked). Release deletes the key immediately. If the timer runs out, Redis fires a keyspace expiration event, which automatically pushes a WebSocket broadcast to refresh the seats.

### Redis Key Schema

| Key Pattern               | Value                      | TTL         |
|---------------------------|----------------------------|-------------|
| `seat:{movieID}:{seatID}` | JSON-encoded `Booking`     | 2 min (held) / none (confirmed) |
| `session:{sessionID}`     | Seat key (`seat:...`)      | 2 min (held) / none (confirmed) |

## Testing

Run the concurrency test (requires a running Redis instance):

```bash
go test ./internal/booking/ -run TestConcurrentBooking_ExactlyOneWins -v
```

This spawns **100,000 goroutines** all racing to book the same seat, verifying that exactly one succeeds and the rest receive `ErrSeatAlreadyBooked`.

## Future Plan

- [ ] **Admin Routes**: Add admin endpoints to create, update, and delete movies and configure seating layouts (rows, seats per row) dynamically instead of hardcoding them.
- [x] **WebSocket Updates**: Replace the 2-second polling with WebSocket connections so seat state changes are pushed to all connected clients instantly.
- [ ] **PostgreSQL Adapter**: Implement a `PostgresStore` behind the existing `BookingStore` interface for durable, queryable storage with transaction support.
- [ ] **User Authentication**: Add proper auth (JWT or session-based) instead of relying on client-generated user IDs.
- [ ] **Showtime Support**: Extend the movie model with showtimes and dates so the same movie can have multiple screenings with independent seat maps.
- [ ] **Payment Integration**: Add a payment step between hold and confirm, with the hold TTL acting as a checkout deadline.
- [ ] **Graceful Error Responses**: Return structured JSON error bodies with HTTP status codes consistently across all handlers.
- [ ] **Configurable Hold Duration**: Make the 2-minute TTL configurable via environment variables or admin settings.
- [ ] **Booking History**: Persist confirmed bookings and expose a `GET /users/{userID}/bookings` endpoint for users to view past reservations.
