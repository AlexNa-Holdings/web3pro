package usb_server

import (
	"errors"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/google/gousb"
	"github.com/rs/zerolog/log"
)

var ctx *gousb.Context

type Connection struct {
	Type        string
	Name        string
	HW_Params   interface{}
	USB_ID      string
	Device      *gousb.Device
	Config      *gousb.Config
	Interface   *gousb.Interface
	EndpointOut *gousb.OutEndpoint
	EndpointIn  *gousb.InEndpoint
}

var connections = []*Connection{}
var connections_mutex = &sync.Mutex{}

func Init() {
	go MessageLoop()
}

func MessageLoop() {
	ctx = gousb.NewContext()
	defer ctx.Close()

	ch := bus.Subscribe("usb")
	enum_ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case msg := <-ch:
			if msg.RespondTo != 0 {
				continue // ignore responses
			}

			switch msg.Type {
			case "list":
				list := bus.B_UsbList_Response{}

				if len(connections) == 0 {
					enumerate()
				}

				for _, conn := range connections {
					list = append(list, bus.B_UsbList_Device{
						USB_ID: GetDeviceID(conn.Device),
						Name:   conn.Name,
						Type:   GetHWType(uint16(conn.Device.Desc.Vendor), uint16(conn.Device.Desc.Product)),
					})

					for _, cfg := range conn.Device.Desc.Configs {
						log.Debug().Msgf("Config: %d ", cfg.Number)
						for _, intf := range cfg.Interfaces {
							log.Debug().Msgf("Interface: %d ", intf.Number)

							for _, alt := range intf.AltSettings {
								log.Debug().Msgf("Alternate: %d ", alt.Alternate)
							}
						}
					}
				}

				msg.Respond(list, nil)
			case "write":
				req, ok := msg.Data.(bus.B_UsbWrite)
				if !ok {
					log.Error().Msg("Invalid message data")
					msg.Respond(nil, bus.ErrInvalidMessageData)
					continue
				}
				conn := GetConnection(req.USB_ID)
				if conn == nil {
					log.Error().Msg("Device not found")
					msg.Respond(nil, errors.New("device not found"))
					continue
				}

				_, err := conn.EndpointOut.Write(req.Data)
				if err != nil {
					log.Error().Err(err).Msg("Error writing to device")
					msg.Respond(nil, err)
					continue
				}
				msg.Respond(nil, nil)
			case "read":
				req, ok := msg.Data.(bus.B_UsbRead)
				if !ok {
					log.Error().Msg("Invalid message data")
					msg.Respond(nil, bus.ErrInvalidMessageData)
					continue
				}
				conn := GetConnection(req.USB_ID)
				if conn == nil {
					log.Error().Msg("Device not found")
					msg.Respond(nil, errors.New("device not found"))
					continue
				}

				data := make([]byte, conn.EndpointIn.Desc.MaxPacketSize)
				n, err := conn.EndpointIn.Read(data)
				if err != nil {
					log.Error().Err(err).Msg("Error reading from device")
					msg.Respond(nil, err)
					continue
				}

				msg.Respond(bus.B_UsbRead_Response{Data: data[:n]}, nil)

			}
		case <-enum_ticker.C:
			enumerate()
		}
	}
}

func GetDeviceID(d *gousb.Device) string {
	return d.String()
}

func GetConnection(id string) *Connection {
	connections_mutex.Lock()
	defer connections_mutex.Unlock()

	for _, conn := range connections {
		if conn.Device.String() == id {
			return conn
		}
	}
	return nil
}

func (c *Connection) Close() {
	if c.Interface != nil {
		c.Interface.Close()
	}
	if c.Config != nil {
		c.Config.Close()
	}
	if c.Device != nil {
		c.Device.Close()
	}
}

func enumerate() {
	connections_mutex.Lock()
	defer connections_mutex.Unlock()

	// Open all devices.
	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return IsSupportedDevice(uint16(desc.Vendor), uint16(desc.Product))
	})

	if err != nil {
		log.Error().Err(err).Msg("enumerate: Error opening devices")

		for _, conn := range connections {
			conn.Close()
		}
		connections = []*Connection{}
		return
	}

	all_found := map[string]bool{}

	for _, dev := range devices {

		sid := GetDeviceID(dev)
		found := false
		for _, conn := range connections {
			if conn.Device.String() == sid {
				all_found[sid] = true
				found = true
				break
			}
		}

		if !found {
			cfg, err := dev.Config(1)
			if err != nil {
				log.Fatal().Msgf("%s.Config(1): %v", dev, err)
				continue
			}

			intf, err := cfg.Interface(0, 0)
			if err != nil {
				cfg.Close()
				log.Fatal().Msgf("%s.DefaultInterface(): %v", dev, err)
				continue
			}

			ep_out, err := intf.OutEndpoint(1)
			if err != nil {
				cfg.Close()
				intf.Close()
				log.Fatal().Msgf("%s.OutEndpoint(1): %v", intf, err)
				continue
			}

			ep_in, err := intf.InEndpoint(1)
			if err != nil {
				cfg.Close()
				intf.Close()
				log.Fatal().Msgf("%s.InEndpoint(1): %v", intf, err)
				continue
			}

			// New device
			t := GetHWType(uint16(dev.Desc.Vendor), uint16(dev.Desc.Product))

			connections = append(connections, &Connection{
				Type:        t,
				Name:        "",
				USB_ID:      sid,
				Device:      dev,
				Config:      cfg,
				Interface:   intf,
				EndpointOut: ep_out,
				EndpointIn:  ep_in,
			})
			all_found[sid] = true

			go func(id string, t string) {
				r := bus.Fetch("signer", "init", bus.B_SignerInit{USB_ID: id, Type: t})
				if r.Error != nil {
					log.Error().Err(r.Error).Msg("Init HW: Error getting signer name")
				} else {
					resp, ok := r.Data.(bus.B_SignerInit_Response)
					if !ok {
						log.Error().Msg("Init HW: Invalid response data")
						return
					}
					GetConnection(id).Name = resp.Name
					GetConnection(id).HW_Params = resp.HW_Params
				}
			}(sid, t)
		}
	}

	// Close all devices that are not found
	for i := 0; i < len(connections); i++ {
		if !all_found[connections[i].Device.String()] {
			connections[i].Close()
			connections = append(connections[:i], connections[i+1:]...)
			i--
		}
	}
}
