## Webhook IRC notifier plugin

Plugin for [IRC-Bot](https://github.com/greboid/irc-bot)

Receives notifications from a URL instance and outputs them to a channel.

 - go build go build github.com/greboid/irc-webhook/v2/cmd/webhook
 - docker run ghcr.io/greboid/irc-webhook
 
#### Configuration

At a bare minimum you also need to give it a channel, a secret to use as part of the URL to receive notifications
 on and an RPC token.  You'll like also want to specify the bot host.
 
 You need to authenticate each request with the "x-api-key" header set to either the admin API key, or a created one

Once configured the following routes are available:
 
 - <boturl>/webhook/keys (only available with the admin key)
 - <boturl/webhook/sendmessage - Takes a payload of json like { "message": "This is a message" }

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
