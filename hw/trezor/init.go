package trezor

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/signer/trezorproto"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func init_trezor(usb_id string) (*Trezor, error) {

	t := &Trezor{
		USB_ID: usb_id,
	}

	kind, reply, err := t.RawCall(&trezorproto.Initialize{})
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error initializing device: %s", usb_id)
		return nil, err
	}
	if kind != trezorproto.MessageType_MessageType_Features {
		log.Error().Msgf("Init: Expected reply type %s, got %s", MessageName(trezorproto.MessageType_MessageType_Features), MessageName(kind))
		return nil, errors.New("trezor: expected reply type " + MessageName(trezorproto.MessageType_MessageType_Features) + ", got " + MessageName(kind))
	}
	features := new(trezorproto.Features)
	err = proto.Unmarshal(reply, features)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error unmarshalling features: %s", usb_id)
		return nil, err
	}

	t.Features = features
	t.Name = t.Features.GetLabel()

	return t, nil
}
