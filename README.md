## Webhook IRC notifier plugin

Plugin for [IRC-Bot](https://github.com/greboid/irc-bot) that relays HTTP webhook requests to IRC channels.

The plugin does not listen for network connections itself. On startup it registers itself with an IRC-bot instance over gRPC, and the bot forwards matching HTTP requests to it. External clients send requests to the IRC-bot's webhook endpoint using the paths described below.

### Configuration

The following configuration settings are supported. Each flag can also be set through the matching environment variable.

| Flag                | Env var            | Default     | Description                                   |
| ------------------- | ------------------ | ----------- | --------------------------------------------- |
| `-rpc-host`         | `RPC_HOST`         | `localhost` | Hostname of the IRC-bot instance              |
| `-rpc-port`         | `RPC_PORT`         | `8001`      | Port to connect to IRC-bot on                 |
| `-rpc-token`        | `RPC_TOKEN`        | -           | Authentication token for IRC-bot              |
| `-channel`          | `CHANNEL`          | -           | Channel to send messages to by default        |
| `-allowed-channels` | `ALLOWED_CHANNELS` | -           | List of channels that messages may be sent to |
| `-debug`            | `DEBUG`            | `false`     | Whether to enable debug logging               |
| `-db-path`          | `DB_PATH`          | `/data/db`  | Path to the token database file               |
| `-admin-key`        | `ADMIN_KEY`        | -           | Admin key for incoming requests (required)    |

The plugin refuses to start if `-admin-key` is not set.

### Authentication

All requests must include an `X-API-Key` header containing either:

- the admin key set via `-admin-key`, or
- a key created through the [keys API](#webhookkeys).

Any valid key may send messages. Only the admin key may manage keys via the keys API.

### API

#### /webhook/sendmessage

Sends a message to IRC. If the message does not specify a channel, the default channel from `-channel` is used.

```http
POST /webhook/sendmessage HTTP/1.1
X-Api-Key: my-key
Content-Type: application/json

{
  "message": "Hello world!",
  "channel": "#some-non-default-channel"
}
```
| Status | Body                   | Condition                          |
| ------ | ---------------------- | ---------------------------------- |
| `200`  | `Delivered`            | Message sent                       |
| `400`  | `Unable to decode`     | Body is not valid JSON             |
| `401`  | `unauthorized`         | Missing, invalid, or non-admin key |
| `401`  | `unauthorized channel` | Channel not in allowed list        |
| `500`  | `Unable to send`       | IRC-bot rejected the message       |

#### /webhook/keys

Manages the API keys permitted to send messages. Requires the admin key; other valid keys are rejected with `400 Unknown`.

A GET request lists all existing keys:
```http
GET /webhook/keys HTTP/1.1
X-Api-Key: my-admin-key
```
```http
HTTP/1.1 200 OK
Content-Type: application/json

["a-key", "another-key"]
```

A POST request adds a new key. The key is taken from the `message` field:
```http
POST /webhook/keys HTTP/1.1
X-Api-Key: my-admin-key
Content-Type: application/json

{"message": "my-new-key"}
```
```http
HTTP/1.1 200 OK
Content-Type: text/plain

User added
```

A DELETE request removes a key, also taken from the `message` field:

```http
DELETE /webhook/keys HTTP/1.1
X-Api-Key: my-admin-key
Content-Type: application/json

{"message": "my-new-key"}
```

```http
HTTP/1.1 200 OK
Content-Type: text/plain

User deleted
```

| Method | Status | Body                    | Condition                    |
| ------ | ------ | ----------------------- | ---------------------------- |
| any    | `400`  | `Unable to decode`      | Body is not valid JSON       |
| any    | `401`  | `unauthorized`          | Key missing or not admin key |
| GET    | `200`  | JSON array of keys      |                              |
| GET    | `500`  | `Unable to get keys`    | Database error               |
| POST   | `200`  | `User added`            | Key created                  |
| POST   | `204`  | `User exists`           | Key already exists           |
| POST   | `400`  | `Key must not be empty` | `message` field is empty     |
| POST   | `500`  | `Unable to add key`     | Database error               |
| DELETE | `200`  | `User deleted`          | Key removed                  |
| DELETE | `404`  | `User not found`        | Key does not exist           |
| DELETE | `500`  | `Unable to delete key`  | Database error               |

Keys are stored in a database file at the path set by `-db-path` and persist across restarts.

#### Other paths

Any other path returns `400` with the body `Unknown`.

### Multi-channel support

By default, all messages are sent to the channel specified in the `channel` flag/env var.

To enable multi-channel support, set `allowed-channels` to a comma-separated list of channels that clients may target (e.g. `#channel1,#channel2,#channel3`). Channel matching is case-insensitive. Setting `allowed-channels` to `*` allows clients to send messages to any channel.