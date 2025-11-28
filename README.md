# Echo Server

An Echo server to demonstrate WebSockets with advanced features.

## Features

### Basic Commands
- **UPPER:** - Converts text to uppercase
  - Example: `UPPER:hello world` → `HELLO WORLD`
- **REVERSE:** - Reverses text
  - Example: `REVERSE:hello` → `olleh`
- **JSON Commands** - Arithmetic operations via JSON
  - Example: `{"command":"add","a":10,"b":5}` → `{"result":15,"command":"add"}`
  - Supported: `add`, `subtract`, `multiply`, `divide`

### Bonus Features

#### 1. Rate Limiting ⏱️
- **Limit**: 10 messages per minute per connection
- Exceeding the limit returns: `{"error":"rate limit exceeded: max 10 messages per minute"}`
- Implementation: Uses a sliding window algorithm tracking message timestamps

#### 2. Command History 📜
- **Command**: Send `HISTORY` to retrieve the last 5 commands
- Returns JSON with command history and count
- Example response:
  ```json
  {
    "history": ["UPPER:test", "REVERSE:hello", "JSON:add"],
    "count": 3
  }
  ```

#### 3. Multi-Client Broadcast 📡
- **Command**: `BROADCAST:your message`
- Sends the message to all connected clients
- Format: `[BROADCAST from <address>] your message`
- Uses a centralized hub with thread-safe connection management

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
