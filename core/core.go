package core

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AlexNa-Holdings/web3pro/wire"
	"github.com/rs/zerolog/log"
)

// Package with "core logic" of device listing
// and dealing with sessions, mutexes, ...
//
// USB package is not imported for efficiency
// reasons - USB package uses imports /usb/lowlevel and
// /usb/lowlevel uses cgo, so it takes about 25 seconds to build;
// so building just this package  on its own
// takes less seconds when we dont import USB
// package and use abstract interfaces instead

// USB* interfaces are implemented in usb package

var USBLog = false

type USBBus interface {
	Enumerate() ([]USBInfo, error)
	Connect(
		path string,
		debug bool, // debug link
		reset bool, // reset is optional, to prevent reseting calls
	) (USBDevice, error)
	Has(path string) bool

	Close() // called on program exit
}

type DeviceType int

const (
	TypeT1Hid        DeviceType = 0
	TypeT1Webusb     DeviceType = 1
	TypeT1WebusbBoot DeviceType = 2
	TypeT2           DeviceType = 3
	TypeT2Boot       DeviceType = 4
	TypeEmulator     DeviceType = 5
)

type USBInfo struct {
	Path      string
	VendorID  int
	ProductID int
	Type      DeviceType
	Debug     bool // has debug enabled?
}

type USBDevice interface {
	io.ReadWriter
	Close(disconnected bool) error
}

type session struct {
	path       string
	id         string
	dev        USBDevice
	call       int32 // atomic
	readMutex  sync.Mutex
	writeMutex sync.Mutex
}

type EnumerateEntry struct {
	Path    string     `json:"path"`
	Vendor  int        `json:"vendor"`
	Product int        `json:"product"`
	Type    DeviceType `json:"-"`     // used only in status page, not in JSON
	Debug   bool       `json:"debug"` // has debug enabled?

	Session      *string `json:"session"`
	DebugSession *string `json:"debugSession"`
}

type EnumerateEntries []EnumerateEntry

func (entries EnumerateEntries) Len() int {
	return len(entries)
}
func (entries EnumerateEntries) Less(i, j int) bool {
	return entries[i].Path < entries[j].Path
}
func (entries EnumerateEntries) Swap(i, j int) {
	entries[i], entries[j] = entries[j], entries[i]
}

type Core struct {
	bus USBBus

	normalSessions sync.Map
	debugSessions  sync.Map
	libusbMutex    sync.Mutex

	allowStealing bool
	reset         bool

	// We cannot make calls and enumeration at the same time,
	// because of some libusb/hidapi issues.
	// However, it is easier to fix it here than in the usb/ packages,
	// because they don't know about whole messages and just send
	// small packets by read/write
	//
	// Those variables help with that
	callsInProgress int
	callMutex       sync.Mutex
	lastInfosMutex  sync.RWMutex
	lastInfos       []USBInfo // when call is in progress, use saved info for enumerating
	// note - both lastInfos and Enumerate result have "paths"
	// as *fake paths* 2.0.26 onwards; just using device IDs,
	// unique for the device

	// the paths we present out are not actual paths
	// from 2.0.26 onwards;
	// it's just an ID of a device
	// we keep the IDs here
	usbPaths  map[int]string // id => path
	biggestID int

	latestSessionID int
}

var (
	ErrWrongPrevSession = errors.New("wrong previous session")
	ErrSessionNotFound  = errors.New("session not found")
	ErrMalformedData    = errors.New("malformed data")
	ErrOtherCall        = errors.New("other call in progress")
)

const (
	VendorT1            = 0x534c
	ProductT1Firmware   = 0x0001
	VendorT2            = 0x1209
	ProductT2Bootloader = 0x53C0
	ProductT2Firmware   = 0x53C1
)

func New(bus USBBus, allowStealing, reset bool) *Core {
	c := &Core{
		bus:           bus,
		allowStealing: allowStealing,
		reset:         reset,
		usbPaths:      make(map[int]string),
	}
	go c.backgroundListen()
	return c
}

const (
	iterMax   = 600
	iterDelay = 500 // ms
)

// This is here just to force recomputing the IDs of
// the disconnected devices (c.usbPaths)
// Note - this does not do anything when no device is connected
// or when no enumerate is run first...
// -> it runs whenever someone calls Enumerate/Listen
// and there are some devices left
// It does not spam USB that much more than listen itself
func (c *Core) backgroundListen() {
	for {
		time.Sleep(iterDelay * time.Millisecond)

		c.lastInfosMutex.RLock()
		linfos := len(c.lastInfos)
		c.lastInfosMutex.RUnlock()
		if linfos > 0 {
			_, err := c.Enumerate()
			if err != nil {
				// we dont really care here
				Trace("error - " + err.Error())
			}
		}
	}
}

func (c *Core) saveUsbPaths(devs []USBInfo) []USBInfo {
	for _, dev := range devs {
		add := true
		for _, usbPath := range c.usbPaths {
			if dev.Path == usbPath {
				add = false
			}
		}
		if add {
			newID := c.biggestID + 1
			c.biggestID = newID
			c.usbPaths[newID] = dev.Path
		}
	}
	reverse := make(map[string]string)
	for id, usbPath := range c.usbPaths {
		discard := true
		for _, dev := range devs {
			if dev.Path == usbPath {
				discard = false
			}
		}
		if discard {
			delete(c.usbPaths, id)
		} else {
			reverse[usbPath] = strconv.Itoa(id)
		}
	}
	res := make([]USBInfo, 0, len(devs))
	for _, dev := range devs {
		res = append(res, USBInfo{
			Path:      reverse[dev.Path],
			VendorID:  dev.VendorID,
			ProductID: dev.ProductID,
			Type:      dev.Type,
			Debug:     dev.Debug,
		})
	}
	return res
}

func (c *Core) Enumerate() ([]EnumerateEntry, error) {
	// avoid enumerating while acquiring the device
	// https://github.com/trezor/trezord-go/issues/221
	c.libusbMutex.Lock()
	defer c.libusbMutex.Unlock()

	// Lock for atomic access to s.callInProgress.  It needs to be over
	// whole function, so that call does not actually start while
	// enumerating.
	c.callMutex.Lock()
	defer c.callMutex.Unlock()

	c.lastInfosMutex.Lock()
	defer c.lastInfosMutex.Unlock()

	// Use saved info if call is in progress, otherwise enumerate.
	infos := c.lastInfos

	Tracef("callsInProgress %d", c.callsInProgress)
	if c.callsInProgress == 0 {
		Trace("bus")
		busInfos, err := c.bus.Enumerate()
		if err != nil {
			return nil, err
		}
		infos = c.saveUsbPaths(busInfos)
		c.lastInfos = infos
	}

	entries := c.createEnumerateEntries(infos)
	Trace("release disconnected")
	c.releaseDisconnected(infos, false)
	c.releaseDisconnected(infos, true)
	return entries, nil
}

func (c *Core) createEnumerateEntry(info USBInfo) EnumerateEntry {
	e := EnumerateEntry{
		Path:    info.Path,
		Vendor:  info.VendorID,
		Product: info.ProductID,
		Type:    info.Type,
		Debug:   info.Debug,
	}
	c.findSession(&e, info.Path, false)
	c.findSession(&e, info.Path, true)
	return e
}

func (c *Core) createEnumerateEntries(infos []USBInfo) EnumerateEntries {
	entries := make(EnumerateEntries, 0, len(infos))
	for _, info := range infos {
		e := c.createEnumerateEntry(info)
		entries = append(entries, e)
	}
	entries.Sort()
	return entries
}

func (entries EnumerateEntries) Sort() {
	sort.Sort(entries)
}

func (c *Core) sessions(debug bool) *sync.Map {
	if debug {
		return &c.debugSessions
	}
	return &c.normalSessions
}

func (c *Core) releaseDisconnected(infos []USBInfo, debug bool) {
	s := c.sessions(debug)
	s.Range(func(k, v interface{}) bool {
		ssid := k.(string)
		ss := v.(*session)
		connected := false
		for _, info := range infos {
			if ss.path == info.Path {
				connected = true
				break
			}
		}
		if !connected {
			Trace(fmt.Sprintf("disconnected device %s", ssid))
			err := c.release(ssid, true, debug)
			// just log if there is an error
			// they are disconnected anyway
			if err != nil {
				Trace(fmt.Sprintf("Error on releasing disconnected device: %s", err))
			}
		}
		return true
	})
}

func (c *Core) Release(session string, debug bool) error {
	return c.release(session, false, debug)
}

func (c *Core) release(
	ssid string,
	disconnected bool,
	debug bool,
) error {
	Trace(fmt.Sprintf("session %s", ssid))
	s := c.sessions(debug)
	v, ok := s.Load(ssid)
	if !ok {
		Trace("session not found")
		return ErrSessionNotFound
	}
	s.Delete(ssid)
	acquired := v.(*session)
	Trace("bus close")
	err := acquired.dev.Close(disconnected)
	return err
}

func (c *Core) Listen(entries []EnumerateEntry, ctx context.Context) ([]EnumerateEntry, error) {
	Trace("start")

	EnumerateEntries(entries).Sort()

	for i := 0; i < iterMax; i++ {
		Trace("before enumerating")
		e, enumErr := c.Enumerate()
		if enumErr != nil {
			return nil, enumErr
		}
		for i := range e {
			e[i].Type = 0 // type is not exported/imported to json
		}
		if reflect.DeepEqual(entries, e) {
			Trace("equal, waiting")
			select {
			case <-ctx.Done():
				Trace(fmt.Sprintf("request closed (%s)", ctx.Err().Error()))
				return nil, nil
			default:
				time.Sleep(iterDelay * time.Millisecond)
			}
		} else {
			Trace("different")
			entries = e
			break
		}
	}
	Trace("encoding and exiting")
	return entries, nil
}

func (c *Core) findPrevSession(path string, debug bool) string {
	s := c.sessions(debug)
	res := ""
	s.Range(func(_, v interface{}) bool {
		ss := v.(*session)
		if ss.path == path {
			res = ss.id
			return false
		}
		return true
	})
	return res
}

func (c *Core) findSession(e *EnumerateEntry, path string, debug bool) {
	s := (c.sessions(debug))
	s.Range(func(_, v interface{}) bool {
		ss := v.(*session)
		if ss.path == path {
			// Copying to prevent overwriting on Acquire and
			// wrong comparison in Listen.
			ssidCopy := ss.id
			if debug {
				e.DebugSession = &ssidCopy
			} else {
				e.Session = &ssidCopy
			}
			return false
		}
		return true
	})
}

func (c *Core) GetDevice(path string) (USBDevice, error) {
	Trace("GetDevice")
	var err error
	s := c.findPrevSession(path, false)

	Trace(fmt.Sprintf("prev session %s", s))

	if s == "" {
		s, err = c.Acquire(path, "", false)
		if err != nil {
			log.Error().Err(err).Msg("GetTrezorName: Error acquiring device")
			return nil, err
		}
	}

	v, ok := c.normalSessions.Load(s)
	if !ok {
		return nil, errors.New("session not found")
	}

	ss := v.(*session)
	return ss.dev, nil
}

func (c *Core) Acquire(
	path, prev string,
	debug bool,
) (string, error) {

	Trace("Acquire")

	// avoid enumerating while acquiring the device
	// https://github.com/trezor/trezord-go/issues/221
	c.libusbMutex.Lock()
	defer c.libusbMutex.Unlock()

	// note - path is *fake path*, basically device ID,
	// because that is what enumerate returns;
	// we convert it to actual path for USB layer

	prevSession := c.findPrevSession(path, debug)

	if prevSession != prev {
		return "", ErrWrongPrevSession
	}

	if (!c.allowStealing) && prevSession != "" {
		return "", ErrOtherCall
	}

	if prev != "" {
		Trace("releasing previous")
		err := c.release(prev, false, debug)
		if err != nil {
			return "", err
		}
	}

	// reset device ONLY if no call on the other port
	// otherwise, USB reset stops other call
	otherSession := c.findPrevSession(path, !debug)
	reset := otherSession == "" && c.reset

	pathI, err := strconv.Atoi(path)
	if err != nil {
		return "", err
	}

	usbPath, exists := c.usbPaths[pathI]
	if !exists {
		return "", errors.New("device not found")
	}

	Trace("trying to connect")
	dev, err := c.tryConnect(usbPath, debug, reset)
	if err != nil {
		return "", err
	}

	id := c.newSession(debug)

	sess := &session{
		path: path,
		dev:  dev,
		call: 0,
		id:   id,
	}

	Trace(fmt.Sprintf("new session is %s", id))

	s := c.sessions(debug)
	s.Store(id, sess)

	return id, nil
}

// Chrome tries to read from trezor immediately after connecting,
// ans so do we.  Bad timing can produce error on s.bus.Connect.
// Try 3 times with a 100ms delay.
func (c *Core) tryConnect(path string, debug bool, reset bool) (USBDevice, error) {
	tries := 0
	for {
		Trace(fmt.Sprintf("try number %d", tries))
		dev, err := c.bus.Connect(path, debug, reset)
		if err != nil {
			if tries < 3 {
				Trace("sleeping")
				tries++
				time.Sleep(100 * time.Millisecond)
			} else {
				Trace("tryConnect - too many times, exiting")
				return nil, err
			}
		} else {
			return dev, nil
		}
	}
}

func (c *Core) newSession(debug bool) string {
	c.latestSessionID++
	res := strconv.Itoa(c.latestSessionID)
	if debug {
		res = "debug" + res
	}
	return res
}

type CallMode int

const (
	CallModeRead      CallMode = 0
	CallModeWrite     CallMode = 1
	CallModeReadWrite CallMode = 2
)

func (c *Core) RawCall(
	body []byte,
	ssid string,
	mode CallMode,
	debug bool,
	ctx context.Context,
) ([]byte, error) {

	c.callMutex.Lock()
	c.callsInProgress++
	c.callMutex.Unlock()

	defer func() {
		c.callMutex.Lock()
		c.callsInProgress--
		c.callMutex.Unlock()
	}()

	s := c.sessions(debug)
	v, ok := s.Load(ssid)
	if !ok {
		return nil, ErrSessionNotFound
	}

	acquired := v.(*session)

	if mode != CallModeWrite {
		// This check is implemented only for /call and /read:
		// - /call: Two /calls should not run concurrently. Otherwise the "message writes" and "message reads"
		//   could interleave and the second caller would read the first response.
		// - /read: Although this could be possible we do not have a use case for that at the moment.
		// The check IS NOT implemented for /post, meaning /post can write even though some /call or /read
		// is in progress (but there are some read/write locks later on).

		Trace("checking other call on same session")
		freeToCall := atomic.CompareAndSwapInt32(&acquired.call, 0, 1)
		if !freeToCall {
			return nil, ErrOtherCall
		}

		Trace("checking other call on same session done")
		defer func() {
			atomic.StoreInt32(&acquired.call, 0)
		}()
	}

	finished := make(chan bool, 1)
	defer func() {
		finished <- true
	}()

	go func() {
		select {
		case <-finished:
			return
		case <-ctx.Done():
			Tracef("detected request close %s, auto-release", ctx.Err().Error())
			errRelease := c.release(ssid, false, debug)
			if errRelease != nil {
				// just log, since request is already closed
				log.Error().Msgf("Error while releasing: %s", errRelease.Error())
			}
		}
	}()

	Trace("before actual logic")
	bytes, err := c.readWriteDev(body, acquired, mode)
	Trace("after actual logic")

	return bytes, err
}

func (c *Core) writeDev(body []byte, device io.Writer) error {
	Trace("decodeRaw")
	msg, err := c.decodeRaw(body)
	if err != nil {
		return err
	}

	Trace("writeTo")
	_, err = msg.WriteTo(device)
	return err
}

func (c *Core) readDev(device io.Reader) ([]byte, error) {
	Trace("readFrom")
	msg, err := wire.ReadFrom(device)
	if err != nil {
		return nil, err
	}

	Trace("encoding back")
	return c.encodeRaw(msg)
}

func (c *Core) readWriteDev(
	body []byte,
	acquired *session,
	mode CallMode,
) ([]byte, error) {

	if mode == CallModeRead {
		if len(body) != 0 {
			return nil, errors.New("non-empty body on read mode")
		}
		Trace("skipping write")
	} else {
		acquired.writeMutex.Lock()
		err := c.writeDev(body, acquired.dev)
		acquired.writeMutex.Unlock()
		if err != nil {
			return nil, err
		}
	}

	if mode == CallModeWrite {
		Trace("skipping read")
		return []byte{0}, nil
	}
	acquired.readMutex.Lock()
	defer acquired.readMutex.Unlock()
	return c.readDev(acquired.dev)
}

func (c *Core) decodeRaw(body []byte) (*wire.Message, error) {
	Trace("readAll")

	Trace("decodeString")

	if len(body) < 6 {
		Trace("body too short")
		return nil, ErrMalformedData
	}

	kind := binary.BigEndian.Uint16(body[0:2])
	size := binary.BigEndian.Uint32(body[2:6])
	data := body[6:]
	if uint32(len(data)) != size {
		Trace("wrong data length")
		return nil, ErrMalformedData
	}

	if wire.Validate(data) != nil {
		Trace("invalid data")
		return nil, ErrMalformedData
	}

	Trace("returning")
	return &wire.Message{
		Kind: kind,
		Data: data,
	}, nil
}

func (c *Core) encodeRaw(msg *wire.Message) ([]byte, error) {
	Trace("start")
	var header [6]byte
	data := msg.Data
	kind := msg.Kind
	size := uint32(len(msg.Data))

	binary.BigEndian.PutUint16(header[0:2], kind)
	binary.BigEndian.PutUint32(header[2:6], size)

	res := append(header[:], data...)

	return res, nil
}

func Tracef(format string, v ...interface{}) {
	if USBLog {
		Tracef(format, v...)
	}
}

func Trace(format string, v ...interface{}) {
	if USBLog {
		Trace(format)
	}
}
