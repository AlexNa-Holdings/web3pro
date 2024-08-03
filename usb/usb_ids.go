package usb

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/google/gousb"
	"github.com/google/gousb/usbid"
	"github.com/rs/zerolog/log"
)

func init_usb_ids() error {
	path := filepath.Join(cmn.DataFolder, USB_IDS)

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		update_usb_ids()
	} else {
		log.Info().Msg("Loading usb.ids")
		file, err := os.Open(path)
		if err != nil {
			log.Error().Err(err).Msg("init_usb_ids: Error opening usb.ids")
			return err
		}
		defer file.Close()

		ids, cls, err := usbid.ParseIDs(file)
		if err != nil {
			log.Error().Err(err).Msg("init_usb_ids: Error parsing usb.ids")
			return err
		}

		usbid.Vendors = ids
		usbid.Classes = cls
		fix_usb_ids()
	}

	log.Trace().Msg("usb.ids loaded")
	return nil
}

func update_usb_ids() error {
	log.Info().Msg("Downloading usb.ids")
	resp, err := http.Get(usbid.LinuxUsbDotOrg)
	if err != nil {
		log.Error().Err(err).Msg("init_usb_ids: Error downloading usb.ids")
		return err
	}
	defer resp.Body.Close()

	// Read the entire file content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("init_usb_ids: Error reading usb.ids")
		return err
	}

	ids, cls, err := usbid.ParseIDs(strings.NewReader(string(content)))
	if err != nil {
		log.Error().Err(err).Msg("init_usb_ids: Error parsing usb.ids")
		return err
	}

	usbid.Vendors = ids
	usbid.Classes = cls
	fix_usb_ids()

	path := filepath.Join(cmn.DataFolder, USB_IDS)
	file, err := os.Create(path)
	if err != nil {
		log.Error().Err(err).Msg("update_usb_ids: Error creating usb.ids")
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		log.Error().Err(err).Msg("update_usb_ids: Error writing usb.ids")
		return err
	}

	log.Trace().Msg("usb.ids updated")
	return nil
}

func fix_usb_ids() error {
	// fix Trezor
	// form https://github.com/trezor/trezor-firmware/blob/main/python/src/trezorlib/models.py

	if usbid.Vendors[gousb.ID(0x1209)] == nil {
		usbid.Vendors[gousb.ID(0x1209)] = &usbid.Vendor{}
	}

	usbid.Vendors[gousb.ID(0x1209)].Name = "SatoshiLabs"
	usbid.Vendors[gousb.ID(0x1209)].Product[gousb.ID(0x53C0)] = &usbid.Product{Name: "Trezor One"}
	usbid.Vendors[gousb.ID(0x1209)].Product[gousb.ID(0x53C1)] = &usbid.Product{Name: "Trezor One"}

	if usbid.Vendors[gousb.ID(0x534C)] == nil {
		usbid.Vendors[gousb.ID(0x534C)] = &usbid.Vendor{}
	}

	usbid.Vendors[gousb.ID(0x534C)].Name = "SatoshiLabs"
	usbid.Vendors[gousb.ID(0x534C)].Product[gousb.ID(0x0001)] = &usbid.Product{Name: "Trezor Model T"}

	// fix Ledger
	// form https://github.com/LedgerHQ/ledger-live/blob/develop/libs/ledgerjs/packages/devices/src/index.ts
	if usbid.Vendors[gousb.ID(0x2c97)] == nil {
		usbid.Vendors[gousb.ID(0x2c97)] = &usbid.Vendor{}
	}
	usbid.Vendors[gousb.ID(0x2c97)].Name = "Ledger"
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x0001)] = &usbid.Product{Name: "Ledger Nano S"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x1001)] = &usbid.Product{Name: "Ledger Nano S"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x1011)] = &usbid.Product{Name: "Ledger Nano S"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x0000)] = &usbid.Product{Name: "Ledger Blue"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x0004)] = &usbid.Product{Name: "Ledger Nano X"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x4000)] = &usbid.Product{Name: "Ledger Nano X"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x4011)] = &usbid.Product{Name: "Ledger Nano X"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x0005)] = &usbid.Product{Name: "Ledger Nano S Plus"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x5000)] = &usbid.Product{Name: "Ledger Nano S Plus"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x5011)] = &usbid.Product{Name: "Ledger Nano S Plus"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x6000)] = &usbid.Product{Name: "Ledger Stax"}
	usbid.Vendors[gousb.ID(0x2c97)].Product[gousb.ID(0x7000)] = &usbid.Product{Name: "Ledger Europa"}

	return nil
}
