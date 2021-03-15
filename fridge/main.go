package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ctison/iot/fridge/fridge"
	"github.com/spf13/cobra"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/keyboard"
	"gobot.io/x/gobot/platforms/mqtt"
)

func main() {
	// Instantiate the command line.
	cmd := cobra.Command{
		Use:                   "fridge CLIENT_ID [TOPIC=/fr/fridge/51966]",
		Args:                  cobra.RangeArgs(1, 2),
		RunE:                  run,
		DisableFlagsInUseLine: true,
		SilenceUsage:          true,
	}

	// Setup the flags.
	cmd.Flags().Bool("help", false, "Print help")
	cmd.Flags().StringP("url", "u", "tls://iot.fr-par.scw.cloud:8883", "Server URL to connect to")
	cmd.Flags().StringP("client-crt", "c", "crt.pem", "Path to client certificate")
	cmd.Flags().StringP("client-key", "k", "key.pem", "Path to client key")
	cmd.Flags().StringP("server-crt", "s", "ca.pem", "Path to server certificate")

	// Execute the command line.
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Instantiates and starts a robot.
func run(cmd *cobra.Command, args []string) error {
	clientID := args[0]
	topic := "/fr/fridge/51966"
	if len(args) > 1 {
		topic = args[1]
	}

	fmt.Println("Topic:", topic)

	clientCrt := cmd.Flag("client-crt").Value.String()
	clientKey := cmd.Flag("client-key").Value.String()
	serverCrt := cmd.Flag("server-crt").Value.String()

	// Instantiate a bot.
	fridge := &Fridge{
		mqtt:  mqtt.NewAdaptor(cmd.Flag("url").Value.String(), clientID),
		keys:  keyboard.NewDriver(),
		topic: topic,
		FridgeModel: fridge.FridgeModel{
			Temp:        ambientTemperature,
			DesiredTemp: 4.,
		},
	}

	// Configure TLS.
	fridge.mqtt.SetClientCert(string(clientCrt))
	fridge.mqtt.SetClientKey(string(clientKey))
	fridge.mqtt.SetServerCert(string(serverCrt))
	fridge.mqtt.SetUseSSL(true)
	fridge.mqtt.SetAutoReconnect(true)

	// Instantiate a robot.
	robot := gobot.NewRobot("bot",
		[]gobot.Connection{fridge.mqtt},
		[]gobot.Device{fridge.keys},
		fridge.reconcile,
	)

	// Start the robot.
	return robot.Start()
}

var (
	ambientTemperature = fridge.FridgeTemperature(25.)
)

// Fridge holds the context for its reconcile method.
type Fridge struct {
	*gobot.Robot       `json:"-"`
	keys               *keyboard.Driver
	mqtt               *mqtt.Adaptor
	topic              string
	fridge.FridgeModel `json:",inline"`
}

// Reconcile the fridge.
func (fridge *Fridge) reconcile() {
	// Handle keyboard events as the extenal events for our fridge.
	_ = fridge.keys.On(keyboard.Key, func(data interface{}) {
		key := data.(keyboard.KeyEvent)
		switch key.Key {
		case keyboard.O:
			fridge.IsDoorOpen = true
		case keyboard.C:
			fridge.IsDoorOpen = false
		case keyboard.ArrowUp:
			fridge.DesiredTemp += 1
		case keyboard.ArrowDown:
			fridge.DesiredTemp -= 1
		case keyboard.I:
			fmt.Println(fridge)
		case keyboard.A:
			// Emulate an alert.
			token, err := fridge.mqtt.PublishWithQOS(fridge.topic+"/alert", 1, []byte("Temperature Warning"))
			if err != nil {
				log.Printf("Error: %v", err)
				break
			}
			token.Wait()
			if err := token.Error(); err != nil {
				log.Printf("Error: %v", err)
			}
		}
	})
	// Emulate fridge environment.
	gobot.Every(1*time.Second, func() {
		if err := fridge.publish(); err != nil {
			fridge.Log("Error: " + err.Error())
		}
		if fridge.IsDoorOpen {
			if fridge.Temp < ambientTemperature {
				fridge.Temp += 1
			}
		} else {
			if fridge.Temp > fridge.DesiredTemp {
				fridge.Temp -= 1
			}
		}
	})
	// Handle alert events.
	_, _ = fridge.mqtt.OnWithQOS(fridge.topic+"/alert", 0, func(msg mqtt.Message) {
		fridge.Log("ALERT: " + string(msg.Payload()))
	})
}

// Publish the state of the fridge.
func (fridge *Fridge) publish() error {
	msg, err := json.Marshal(fridge)
	if err != nil {
		return err
	}
	log.Println(string(msg))
	token, err := fridge.mqtt.PublishWithQOS(fridge.topic, 1, msg)
	if err != nil {
		return err
	}
	token.Wait()
	return token.Error()
}

// Log messages where the user can see them.
func (fridge *Fridge) Log(msg string) {
	log.Println(msg)
}

func (fridge *Fridge) String() string {
	isDoorOpen := map[bool]string{true: "open", false: "closed"}
	return fmt.Sprintf("Fridge temperature is %vºC and door is %s.", fridge.Temp, isDoorOpen[fridge.IsDoorOpen])
}
