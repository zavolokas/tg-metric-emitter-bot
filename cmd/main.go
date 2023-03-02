package main

import (
	"context"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	log "github.com/sirupsen/logrus"
)

const (
	exitCodeErr       = 1
	exitCodeInterrupt = 2
)

func main() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	err := godotenv.Load()
	if err != nil {
		log.WithError(err).Fatal("Error loading .env file")
		return
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	go func() {
		select {
		case <-signalChan: // first signal, cancel context
			cancel()
		case <-ctx.Done():
		}
		<-signalChan // second signal, hard exit
		os.Exit(exitCodeInterrupt)
	}()

	token := os.Getenv("INFLUXDB_TOKEN")
	// Set up a connection to InfluxDB
	client := influxdb2.NewClient("http://localhost:8086", token)
	defer client.Close()

	botToken := os.Getenv("ME_TGBOT_TOKEN")
	bot, err := telego.NewBot(botToken, telego.WithLogger(log.StandardLogger()))
	if err != nil {
		log.WithError(err).Fatal("failed to create bot")
		os.Exit(1)
	}

	// Get updates channel
	updates, _ := bot.UpdatesViaLongPolling(nil)

	// Stop getting updates
	defer bot.StopLongPolling()

	for update := range updates {

		log.WithFields(log.Fields{
			"update": update,
		}).Debug("got message")

		msg := ""
		var chatID int64
		if update.Message != nil {
			log.WithFields(log.Fields{
				"update_msg": update.Message,
			}).Debug("update.Message != nil")
			msg = update.Message.Text
			chatID = update.Message.Chat.ID
		}
		if len(msg) == 0 && update.ChannelPost != nil {
			log.WithFields(log.Fields{
				"update_msg": update.ChannelPost,
			}).Debug("update.ChannelPost != nil")
			msg = update.ChannelPost.Text
			chatID = update.ChannelPost.Chat.ID
		}

		log.WithFields(log.Fields{
			"text_msg": msg,
			"chat_id":  chatID,
		}).Debug("what we have here?")

		if !strings.HasPrefix(msg, "/emit") || !strings.Contains(msg, "@metricemitter_bot") {
			continue
		}

		regPattern := `\/emit\s+(\w+)\s+(\w+)\s((?:January|February|March|April|May|June|July|August|September|October|November|December)\s\d{1,2},\s\d{4}\s+at\s+(?:[01]\d|2[0-3]):[0-5]\d(?:AM|PM))`
		re := regexp.MustCompile(regPattern)
		matches := re.FindStringSubmatch(msg)
		log.WithFields(log.Fields{
			"matches": matches,
		}).Debug("matches")

		if len(matches) != 4 {
			log.Info("wrong command format, please use /emit <metric_name> <field> <metric_timestamp>")
			_, _ = bot.SendMessage(tu.Message(
				tu.ID(chatID),
				"wrong command format, please use /emit <metric_name> <field> <metric_timestamp>",
			))
			continue
		}
		log.Debug("command looks good!")

		metricName := matches[1]
		field := matches[2]
		timeStamp, err := time.Parse("January 2, 2006 at 3:04PM", matches[3])
		if err != nil {
			log.WithError(err).Error("failed to parse time")
			_, _ = bot.SendMessage(tu.Message(
				tu.ID(chatID),
				"failed to parse time",
			))
			continue
		}

		log.WithFields(log.Fields{
			"metric_name": metricName,
			"field":       field,
			"time_stamp":  timeStamp.Format(time.UnixDate),
			"chat_id":     chatID,
			"message":     msg,
		}).Info("got message")

		writeAPI := client.WriteAPI("private", "default")

		// create point using fluent style
		p := influxdb2.NewPointWithMeasurement(metricName).
			AddTag("camera", field).
			SetTime(time.Now()).
			AddField("activate", 1)

		// write point asynchronously
		writeAPI.WritePoint(p)
		// Flush writes
		writeAPI.Flush()

		// Send message
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(chatID),
			"metric has been emitted!",
		))

	}
}
