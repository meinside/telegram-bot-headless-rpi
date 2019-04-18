package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/meinside/rpi-tools/hardware"
	"github.com/meinside/rpi-tools/status"

	bot "github.com/meinside/telegram-bot-go"
)

// constants
const (
	LocationLivePeriodSeconds = 60

	ConfigFilename = "config.json"
)

// commands and messages
const (
	CommandStart    = "/start"
	CommandStatus   = "/status"
	CommandLocation = "/where"
	CommandReboot   = "/reboot"
	CommandShutdown = "/shutdown"
	CommandHelp     = "/help"
	CommandCancel   = "/cancel"

	MessageConfirmReboot   = "Really reboot?"
	MessageConfirmShutdown = "Really shutdown?"
	MessageRebooting       = "Rebooting..."
	MessageShuttingdown    = "Shutting down..."
	MessageHelp            = `Usage:

/status  : Show current status of your Raspberry Pi.
/where   : Show current location of your Raspberry Pi. (based on external IP)
/reboot  : Reboot your Raspberry Pi.
/shutdown: Shutdown your Raspberry Pi.
/help    : Show this help message.
`
	MessageCanceled = "Canceled."
	MessageError    = "Error."

	ButtonYes    = "Yes"
	ButtonCancel = "Cancel"
)

// variables
var apiToken string
var availableIds []string
var telegramMonitorInterval uint
var isVerbose bool
var allKeyboards [][]bot.KeyboardButton

// struct for config file
type config struct {
	APIToken                string   `json:"api_token"`
	AvailableIds            []string `json:"available_ids"`
	TelegramMonitorInterval uint     `json:"telegram_monitor_interval"` // in sec
	IsVerbose               bool     `json:"is_verbose"`
}

// Read config
func getConfig() (conf config, err error) {
	var execFilepath string
	if execFilepath, err = os.Executable(); err == nil {
		var file []byte
		if file, err = ioutil.ReadFile(filepath.Join(filepath.Dir(execFilepath), ConfigFilename)); err == nil {
			var cfg config
			if err = json.Unmarshal(file, &cfg); err == nil {
				return cfg, nil
			}
		}
	}

	return config{}, err
}

func init() {
	// read variables from config file
	if conf, err := getConfig(); err == nil {
		apiToken = conf.APIToken
		availableIds = conf.AvailableIds
		telegramMonitorInterval = conf.TelegramMonitorInterval
		isVerbose = conf.IsVerbose

		// all keyboard buttons
		allKeyboards = [][]bot.KeyboardButton{
			[]bot.KeyboardButton{
				bot.KeyboardButton{
					Text: CommandStatus,
				},
				bot.KeyboardButton{
					Text: CommandLocation,
				},
				bot.KeyboardButton{
					Text: CommandHelp,
				},
			},
			[]bot.KeyboardButton{
				bot.KeyboardButton{
					Text: CommandReboot,
				},
				bot.KeyboardButton{
					Text: CommandShutdown,
				},
			},
		}
	} else {
		panic(err.Error())
	}
}

// check if given Telegram id is available
func isAvailableID(id string) bool {
	for _, v := range availableIds {
		if v == id {
			return true
		}
	}
	return false
}

// for processing an incoming update from Telegram
func processUpdate(b *bot.Bot, update bot.Update) bool {
	// check username
	var userID string
	if update.Message.From.Username == nil {
		log.Printf("*** Not allowed (no user name): %s", update.Message.From.FirstName)
		return false
	}
	userID = *update.Message.From.Username
	if !isAvailableID(userID) {
		log.Printf("*** Id not allowed: %s", userID)
		return false
	}

	if update.HasMessage() {
		txt := *update.Message.Text

		if isVerbose {
			log.Printf("received telegram message: %s", txt)
		}

		options := defaultOptions()

		// 'is typing...'
		b.SendChatAction(update.Message.Chat.ID, bot.ChatActionTyping)

		if strings.HasPrefix(txt, CommandStart) {
			message := MessageHelp

			// send message
			if sent := b.SendMessage(update.Message.Chat.ID, message, options); !sent.Ok {
				log.Printf("*** Failed to send message: %s", *sent.Description)
			}
		} else if strings.HasPrefix(txt, CommandStatus) {
			hostname, _ := status.Hostname()
			intIPs := strings.Join(status.IpAddresses(), ", ")
			extIP, _ := status.ExternalIpAddress()
			uptime, _ := status.Uptime()
			freeSpaces, _ := status.FreeSpaces()

			status := fmt.Sprintf(
				`Hostname : %s

Internal IP : %s

External IP : %s

Uptime :
%s

Free Spaces :
%s`,
				hostname,
				intIPs,
				extIP,
				uptime,
				freeSpaces,
			)

			if sent := b.SendMessage(update.Message.Chat.ID, status, options); !sent.Ok {
				log.Printf("*** Failed to send message: %s", *sent.Description)
			}
		} else if strings.HasPrefix(txt, CommandLocation) {
			if extIP, err := status.ExternalIpAddress(); err == nil {
				if geoInfo, err := status.GeoLocation(extIP); err == nil {
					options["live_period"] = LocationLivePeriodSeconds

					if sent := b.SendLocation(
						update.Message.Chat.ID,
						geoInfo.Location.Latitude,
						geoInfo.Location.Longitude,
						options,
					); !sent.Ok {
						log.Printf("*** Failed to send location: %s", *sent.Description)
					}
				} else {
					if sent := b.SendMessage(
						update.Message.Chat.ID,
						fmt.Sprintf("Failed to get geo location: %s", err),
						options,
					); !sent.Ok {
						log.Printf("*** Failed to send message: %s", *sent.Description)
					}
				}
			} else {
				if sent := b.SendMessage(
					update.Message.Chat.ID,
					fmt.Sprintf("Failed to get external ip address: %s", err),
					options,
				); !sent.Ok {
					log.Printf("*** Failed to send message: %s", *sent.Description)
				}
			}
		} else if strings.HasPrefix(txt, CommandReboot) {
			// inline keyboards
			reboot := CommandReboot
			cancel := CommandCancel
			buttons := [][]bot.InlineKeyboardButton{
				[]bot.InlineKeyboardButton{
					bot.InlineKeyboardButton{
						Text:         ButtonYes,
						CallbackData: &reboot,
					},
					bot.InlineKeyboardButton{
						Text:         ButtonCancel,
						CallbackData: &cancel,
					},
				},
			}

			// options
			options["reply_markup"] = bot.InlineKeyboardMarkup{
				InlineKeyboard: buttons,
			}

			// send message
			message := MessageConfirmReboot
			if sent := b.SendMessage(update.Message.Chat.ID, message, options); !sent.Ok {
				log.Printf("*** Failed to send message: %s", *sent.Description)
			}
		} else if strings.HasPrefix(txt, CommandShutdown) {
			// inline keyboards
			shutdown := CommandShutdown
			cancel := CommandCancel
			buttons := [][]bot.InlineKeyboardButton{
				[]bot.InlineKeyboardButton{
					bot.InlineKeyboardButton{
						Text:         ButtonYes,
						CallbackData: &shutdown,
					},
					bot.InlineKeyboardButton{
						Text:         ButtonCancel,
						CallbackData: &cancel,
					},
				},
			}

			// options
			options["reply_markup"] = bot.InlineKeyboardMarkup{
				InlineKeyboard: buttons,
			}

			// send message
			message := MessageConfirmShutdown
			if sent := b.SendMessage(update.Message.Chat.ID, message, options); !sent.Ok {
				log.Printf("*** Failed to send message: %s", *sent.Description)
			}
		} else if strings.HasPrefix(txt, CommandHelp) {
			// send message
			message := MessageHelp
			if sent := b.SendMessage(update.Message.Chat.ID, message, options); !sent.Ok {
				log.Printf("*** Failed to send message: %s", *sent.Description)
			}
		} else {
			message := fmt.Sprintf("No such command: %s", txt)

			log.Println(message)

			// send message
			if sent := b.SendMessage(update.Message.Chat.ID, message, options); !sent.Ok {
				log.Printf("*** Failed to send message: %s", *sent.Description)
			}
		}
	}

	return false
}

// process incoming callback query
func processCallbackQuery(b *bot.Bot, update bot.Update) bool {
	// process result
	result := false

	query := *update.CallbackQuery
	txt := *query.Data

	var message string = MessageError
	if strings.HasPrefix(txt, CommandCancel) {
		message = MessageCanceled
	} else {
		if strings.HasPrefix(txt, CommandReboot) {
			message = MessageRebooting
		} else if strings.HasPrefix(txt, CommandShutdown) {
			message = MessageShuttingdown
		}
	}

	// answer callback query
	if apiResult := b.AnswerCallbackQuery(query.ID, map[string]interface{}{"text": message}); apiResult.Ok {
		// edit message and remove inline keyboards
		options := map[string]interface{}{
			"chat_id":    query.Message.Chat.ID,
			"message_id": query.Message.MessageID,
		}
		if apiResult := b.EditMessageText(message, options); apiResult.Ok {
			result = true

			if message == MessageRebooting {
				// reboot
				if output, err := hardware.RebootNow(); err != nil {
					if sent := b.SendMessage(update.Message.Chat.ID, output, options); !sent.Ok {
						log.Printf("*** Failed to send message: %s", *sent.Description)
					}
				}
			} else if message == MessageShuttingdown {
				// shutdown
				if output, err := hardware.ShutdownNow(); err != nil {
					if sent := b.SendMessage(update.Message.Chat.ID, output, options); !sent.Ok {
						log.Printf("*** Failed to send message: %s", *sent.Description)
					}
				}
			} else if message != MessageCanceled {
				log.Printf("*** Unprocessable callback query message: %s", message)
			}
		} else {
			log.Printf("*** Failed to edit message text: %s", *apiResult.Description)
		}
	} else {
		log.Printf("*** Failed to answer callback query: %+v", query)
	}

	return result
}

// returns default options for messages
func defaultOptions() map[string]interface{} {
	return map[string]interface{}{
		"reply_markup": bot.ReplyKeyboardMarkup{
			Keyboard:       allKeyboards,
			ResizeKeyboard: true,
		},
	}
}

func main() {
	client := bot.NewClient(apiToken)
	client.Verbose = isVerbose

	// monitor for new telegram updates
	if me := client.GetMe(); me.Ok { // get info about this bot
		log.Printf("Launching bot: @%s (%s)", *me.Result.Username, me.Result.FirstName)

		// delete webhook (getting updates will not work when wehbook is set up)
		if unhooked := client.DeleteWebhook(); unhooked.Ok {
			// wait for new updates
			client.StartMonitoringUpdates(0, int(telegramMonitorInterval), func(b *bot.Bot, update bot.Update, err error) {
				if err == nil {
					if update.HasMessage() {
						processUpdate(b, update)
					} else if update.HasCallbackQuery() {
						processCallbackQuery(b, update)
					}
				} else {
					log.Printf("*** Error while receiving update (%s)", err.Error())
				}
			})
		} else {
			panic("Failed to delete webhook")
		}
	} else {
		panic("Failed to get info of the bot")
	}
}
