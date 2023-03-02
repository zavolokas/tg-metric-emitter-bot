Influx password reset

```bash

# login into influx container
docker exec -it <container id> bash
cat ./var/lib/influxdb2/influxd.bolt

# find there admin token and use it to reset password
docker exec -it <container id> influx user password -n admin -t <token>

```

To add annotations the following query can be used
```
from(bucket: "default")
  |> range(start: v.timeRangeStart, stop:v.timeRangeStop)
  |> filter(fn: (r) =>
    r._measurement == "events" and
    (r._field == "text" or
    r._field == "title")
  )
```

# Telegram Bots

## Webhook

Send a message to the bot to get the chat id

```bash
# set bot token
WH_TGBOT_TOKEN=

# get chat id
CHAT_ID=$(curl -X GET "https://api.telegram.org/bot${WH_TGBOT_TOKEN}/getUpdates" | jq -r '.result[0].channel_post.sender_chat.id')


curl -X POST "https://api.telegram.org/bot${WH_TGBOT_TOKEN}/sendMessage" \
    -H 'Content-Type: application/json' \
    -d '{"chat_id":"${CHAT_ID}", "text":"djsakda"}'
    # -d "{'chat_id':'${CHAT_ID}', 'text':'djsakda'}"
```
