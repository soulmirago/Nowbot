BOT_BINARY=bot

JS_FILES = $(shell find static/src/ -type f -name '*.js')

.PHONY: all
all: bot

bot: cmd/bot/bot.go
	go build -o ${BOT_BINARY} cmd/bot/bot.go

npm: static/package.json
	cd static && npm install .

gulp: $(JS_FILES)
	cd static && gulp dist

.PHONY: static
static: npm gulp

.PHONY: clean
clean:
	rm -r ${BOT_BINARY} ${WEB_BINARY} static/dist/
