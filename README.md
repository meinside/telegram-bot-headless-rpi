# telegram-bot-headless-rpi

This is a Telegram Bot for controlling Raspberry Pi headlessly.

## Install

Get it:

```bash
$ go get -d github.com/meinside/telegram-bot-headless-rpi
```

and build it:

```bash
$ cd $GOPATH/src/github.com/meinside/telegram-bot-headless-rpi
$ go build
```

## Configuration

### Config file

```bash
$ cd $GOPATH/src/github.com/meinside/telegram-bot-headless-rpi
$ cp config.json.sample config.json
$ vi config.json
```

and edit values to yours:

```
{
	"api_token": "0123456789:abcdefghijklmnopqrstuvwyz-x-0a1b2c3d4e",
	"available_ids": [
		"telegram_id_1",
		"telegram_id_2",
		"telegram_id_3"
	],
	"telegram_monitor_interval": 1,
	"ipstack_access_key": "xyz_123466789_abcd",
	"ipstack_premium": false,
	"is_verbose": false
}
```

### systemd

```bash
$ sudo cp systemd/telegram-bot-headless-rpi.service /lib/systemd/system/
$ sudo vi /lib/systemd/system/telegram-bot-headless-rpi.service
```

and edit **User**, **Group**, **WorkingDirectory** and **ExecStart** values.

It will launch automatically on boot with:

```bash
$ sudo systemctl enable telegram-bot-headless-rpi.service
```

and will start with:

```bash
$ sudo systemctl start telegram-bot-headless-rpi.service
```

## Send commands through Telegram

![capture](https://user-images.githubusercontent.com/185988/37770794-ecb0440c-2e18-11e8-8486-02b992469f93.png)

## License

MIT

