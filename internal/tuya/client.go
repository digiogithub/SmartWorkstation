// Package tuya connects to the Tuya IoT Cloud via the official
// tuya-connector-go SDK (Pulsar-based event streaming) and triggers WoL when
// the virtual smart switch is turned on.
//
// Message flow:
//   User taps switch in Tuya app
//     → Tuya cloud updates virtual-device state
//     → Pulsar event "statusReport" fires
//     → our handler receives StatusReportMessage
//     → if switch_1 == true (rising edge): send WoL magic packet
//
// Virtual device setup: see docs/tuya-setup.md.
package tuya

import (
	"context"
	"log"

	"github.com/digiogithub/smartworkstation/internal/config"
	"github.com/digiogithub/smartworkstation/internal/wol"

	// Importing connector triggers blank-import side-effects that register
	// the header, token, sign, message and logger extensions.
	"github.com/tuya/tuya-connector-go/connector"
	"github.com/tuya/tuya-connector-go/connector/constant"
	"github.com/tuya/tuya-connector-go/connector/env"
	"github.com/tuya/tuya-connector-go/connector/env/extension"
	"github.com/tuya/tuya-connector-go/connector/httplib"
	"github.com/tuya/tuya-connector-go/connector/message/event"
)

// Client manages the Tuya Pulsar subscription and WoL dispatch.
type Client struct {
	cfg         *config.Config
	switchState bool // tracks last known switch value to detect rising edge
}

// NewClient initialises the Tuya SDK with project credentials and region.
// Call Start to begin receiving events.
func NewClient(cfg *config.Config) (*Client, error) {
	apiHost, msgHost := regionEndpoints(cfg.Tuya.Region)

	connector.InitWithOptions(
		env.WithApiHost(apiHost),
		env.WithMsgHost(msgHost),
		env.WithAccessID(cfg.Tuya.ClientID),
		env.WithAccessKey(cfg.Tuya.ClientSecret),
		env.WithDebugMode(cfg.Tuya.DebugEvents),
	)

	return &Client{cfg: cfg}, nil
}

// Start subscribes to Tuya Pulsar events and runs until ctx is cancelled.
// The function returns immediately; processing happens in background goroutines.
func (c *Client) Start(ctx context.Context) error {
	msg := extension.GetMessage(constant.TUYA_MESSAGE)

	// Open the Pulsar consumer.
	msg.InitMessageClient()

	// Register typed handlers — the SDK uses reflection to dispatch by type.
	msg.SubEventMessage(func(m *event.StatusReportMessage) {
		c.onStatusReport(m)
	})
	msg.SubEventMessage(func(m *event.OnlineMessage) {
		if m.DevID == c.cfg.Tuya.DeviceID {
			log.Printf("[tuya] device online")
		}
	})
	msg.SubEventMessage(func(m *event.OfflineMessage) {
		if m.DevID == c.cfg.Tuya.DeviceID {
			log.Printf("[tuya] device offline")
		}
	})

	log.Printf("[tuya] Pulsar listener started — region=%s  device=%s  dp=%s",
		c.cfg.Tuya.Region, c.cfg.Tuya.DeviceID, c.cfg.Tuya.SwitchDP)

	// Stop cleanly when the application context is cancelled.
	go func() {
		<-ctx.Done()
		msg.Stop()
		log.Println("[tuya] listener stopped")
	}()

	return nil
}

// onStatusReport handles device status-change events.
func (c *Client) onStatusReport(m *event.StatusReportMessage) {
	if m.DevID != c.cfg.Tuya.DeviceID {
		return
	}

	if c.cfg.Tuya.DebugEvents {
		log.Printf("[tuya] statusReport devId=%s", m.DevID)
		for _, dp := range m.Status {
			log.Printf("[tuya]   dp: code=%s value=%v", dp.Code, dp.Value)
		}
	}

	for _, dp := range m.Status {
		if dp.Code != c.cfg.Tuya.SwitchDP {
			continue
		}

		// The SDK decodes JSON booleans as bool, numbers as float64, strings as string.
		switchOn, ok := dp.Value.(bool)
		if !ok {
			log.Printf("[tuya] DP %s: expected bool, got %T (%v)", dp.Code, dp.Value, dp.Value)
			return
		}

		log.Printf("[tuya] switch %s → %v", dp.Code, switchOn)

		// Send WoL on the rising edge only to avoid repeated packets.
		if switchOn && !c.switchState {
			c.sendWoL()
		}
		c.switchState = switchOn
	}
}

// sendWoL dispatches the WoL magic packet to the configured target.
func (c *Client) sendWoL() {
	w := c.cfg.WoL
	log.Printf("[wol] sending magic packet → %s  broadcast=%s:%d  repeat=%d",
		w.MACAddress, w.BroadcastIP, w.Port, w.Repeat)

	if err := wol.Send(w.MACAddress, w.BroadcastIP, w.Port, w.Repeat); err != nil {
		log.Printf("[wol] error: %v", err)
	} else {
		log.Printf("[wol] magic packet sent successfully")
	}
}

// regionEndpoints maps a Tuya region code to its REST API and Pulsar hosts.
// Values are taken from the httplib constants to stay in sync with the SDK.
func regionEndpoints(region string) (apiHost, msgHost string) {
	switch region {
	case "us":
		return httplib.URL_US, httplib.MSG_US
	case "cn":
		return httplib.URL_CN, httplib.MSG_CN
	case "in":
		return httplib.URL_IN, httplib.MSG_IN
	default: // "eu"
		return httplib.URL_EU, httplib.MSG_EU
	}
}
