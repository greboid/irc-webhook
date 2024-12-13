## Webhook IRC notifier plugin

Plugin for [IRC-Bot](https://github.com/greboid/irc-bot)

Receives notifications from a URL instance and outputs them to a channel.

- go build github.com/greboid/irc-webhook/v4/cmd/webhook
- docker run ghcr.io/greboid/irc-webhook

### Configuration

The following configuration settings are supported:

| Flag                | Env var            | Default     | Description                                   |
|---------------------|--------------------|-------------|-----------------------------------------------|
| `-rpc-host`         | `RPC_HOST`         | `localhost` | Hostname of the IRC-bot instance              |
| `-rpc-port`         | `RPC_PORT`         | `8001`      | Port to connect to IRC-bot on                 |
| `-rpc-token`        | `RPC_TOKEN`        | -           | Authentication token for IRC-bot              |
| `-channel`          | `CHANNEL`          | -           | Channel to send messages to by default        |
| `-allowed-channels` | `ALLOWED_CHANNELS` | -           | List of channels that messages may be sent to |
| `-debug`            | `DEBUG`            | `false`     | Whether to enable debug logging               |
| `-db-path`          | `DB_PATH`          | `/data/db`  | Path to store token database                  |
| `-admin-key`        | `ADMIN_KEY`        | -           | Default key for incoming requests             |

#### Multi-channel support

By default, all messages will be sent to the channel specified in the `channel`
flag/env var.

To enable multi-channel support, set `allowed-channels` to a comma-separated
list of additional channels to allow (e.g. `#channel1,#channel2,#channel3`).
You can also set `allowed-channels` to `*` to allow clients to send messages
anywhere.

### API

The plugin exposes two endpoints.

All requests must be authenticated with an `X-API-Key` header containing either
the admin key specified in the config, or a key created using the keys API.

#### /webhook/sendmessage

Sends a message to IRC. If allowed channels is specified in the config, you
can specify the channel as well.

```http
POST /webhooks/sendmessage HTTP/1.1
X-Api-Key: my-key
Content-Type: application/json

{
  "message": "Hello world!",
  "channel": "#some-non-default-channel"
}
```

```http
HTTP/1.1 200 OK
Content-Type: text/plain

Delivered
```

Responds with a 200 OK when the message is successfully delivered

#### /webhook/keys

Manages the API keys permitted to interact with the plugin. Note that any key
can use this API, so in effect all keys are "admin" keys.

A GET request will list all existing keys:

```http
GET /webhooks/keys HTTP/1.1
X-Api-Key: my-admin-key
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

["a-key", "another-key"]
```

A POST request with the key in the "message" field will add a new key:

```http
POST /webhooks/keys HTTP/1.1
X-Api-Key: my-admin-key
Content-Type: application/json

{"message": "my-new-key"}
```

```http
HTTP/1.1 200 OK
Content-Type: text/plain

User added
```

A DELETE request with the key in the "message" field will delete the key:

```http
DELETE /webhooks/keys HTTP/1.1
X-Api-Key: my-admin-key
Content-Type: application/json

{"message": "my-new-key"}
```

```http
HTTP/1.1 200 OK
Content-Type: text/plain

User deleted
```

#### Example running

```
---
version: "3.5"
service:
  goplum:
    image: ghcr.io/greboid/irc-webhook
    environment:
      RPC_HOST: bot
      RPC_TOKEN: <as configured on the bot>
      CHANNEL: #spam
      DB_PATH: /data/db
      ADMIN_KEY: CGP9NDXs
```

```
webhook -rpc-host bot -rpc-token <as configured on the bot> -channel #spam -db-path /data/db -admin-key CGP9NDXs
```
