BOT_BINARY=bot

.PHONY: all
all: bot

bot: cmd/bot/bot.go
	go build -o ${BOT_BINARY} cmd/bot/bot.go

.PHONY: clean
clean:
	rm -r ${BOT_BINARY} static/dist/
