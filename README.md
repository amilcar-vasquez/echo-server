# Echo Server

An Echo server to demonstrate WebSockets with advanced features.

## Features

### Part 1: Uppercase Echo
Converts text to uppercase when message starts with `UPPER:` prefix.
- Example: `UPPER:hello world` → `HELLO WORLD`
- Implementation: Uses `strings.HasPrefix` and `strings.ToUpper`

### Part 2: Reverse Echo
Reverses text when message starts with `REVERSE:` prefix.
- Example: `REVERSE:hello` → `olleh`
- Implementation: Converts to runes for proper Unicode handling, swaps elements from both ends

### Part 3: Broadcast Counter
Tracks total messages received across all connections and includes count in each response.
- Example: Client sends `hello` → Server echoes `#5 hello`
- Implementation: Uses `sync/atomic` for thread-safe counter increments

### Part 4: JSON Command Processing
Accepts JSON commands for arithmetic operations and responds with JSON results.
- Example: `{"command":"add","a":10,"b":5}` → `{"result":15,"command":"add"}`
- Supported operations: `add`, `subtract`, `multiply`, `divide`
- Implementation: Unmarshals JSON, processes command via switch statement, marshals response

### Bonus Challenges

#### Challenge 1: Rate Limiting
Limits each connection to maximum 10 messages per minute.
- Exceeding limit returns: `{"error":"rate limit exceeded: max 10 messages per minute"}`
- Implementation: Per-connection `RateLimiter` with sliding window algorithm using timestamp slice

#### Challenge 2: Command History
Stores last 5 commands per connection, retrievable via `HISTORY` command.
- Returns: `{"history":["UPPER:test","REVERSE:hello"],"count":2}`
- Implementation: Per-connection circular buffer with mutex protection

#### Challenge 3: Multi-Client Broadcast
Sends message to all connected clients when prefixed with `BROADCAST:`.
- Format: `[BROADCAST from 127.0.0.1:12345] your message`
- Implementation: Centralized `ClientHub` with channel-based communication and thread-safe connection map

## Running the Server

```bash
make run
# or
go run cmd/web/main.go
```

Open `web/test.html` in your browser to test all features.

## Testing

- **Rate Limit**: Click "Test Rate Limit" to send 15 rapid messages
- **History**: Use various commands, then click "Get History"
- **Broadcast**: Open multiple browser tabs and test broadcasting between them
