package cbyge

import (
	"encoding/binary"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/unixpickle/essentials"
)

const DefaultTimeout = time.Second * 10

type ControllerDeviceStatus struct {
	StatusPaginatedResponse

	// If IsOnline is false, all other fields are invalid.
	// This means that the device could not be reached.
	IsOnline bool
}

type ControllerDevice struct {
	deviceID string
	switchID uint64
	name     string

	lastStatus     ControllerDeviceStatus
	lastStatusLock sync.RWMutex
}

// DeviceID gets a unique identifier for the device.
func (c *ControllerDevice) DeviceID() string {
	return c.deviceID
}

// Name gets the user-assigned name of the device.
func (c *ControllerDevice) Name() string {
	return c.name
}

// LastStatus gets the last known status of the device.
//
// This is not updated automatically, but it will be updated on a device
// object when Controller.DeviceStatus() is called.
func (c *ControllerDevice) LastStatus() ControllerDeviceStatus {
	c.lastStatusLock.RLock()
	defer c.lastStatusLock.RUnlock()
	return c.lastStatus
}

func (c *ControllerDevice) hasSwitch() bool {
	return c.switchID&0xffffffff == c.switchID
}

func (c *ControllerDevice) isSwitch(id uint32) bool {
	return c.hasSwitch() && uint32(c.switchID) == id
}

func (c *ControllerDevice) deviceIndex() int {
	parsed, _ := strconv.ParseUint(c.deviceID, 10, 64)
	return int(parsed % 1000)
}

// A Controller is a high-level API for manipulating C by GE devices.
type Controller struct {
	sessionInfoLock sync.RWMutex
	sessionInfo     *SessionInfo
	timeout         time.Duration

	// Each device has a list of switches which can reach it, and
	// a current index into this list which is incremented round-robin
	// every time reaching the device results in an error.
	switchMappingLock sync.RWMutex
	switches          map[string][]uint32
	switchIndices     map[string]int

	// Prevent multiple PacketConns at once, since the server boots
	// off one connection when anoher is made.
	packetConnLock sync.Mutex

	// We continually increment our sent sequence ID.
	seqIDLock sync.Mutex
	seqID     uint16
}

// NewController creates a Controller using a pre-created session and a
// specified timeout.
//
// If timeout is 0, then DefaultTimeout is used.
func NewController(s *SessionInfo, timeout time.Duration) *Controller {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	return &Controller{
		sessionInfo: s,
		timeout:     timeout,

		switches:      map[string][]uint32{},
		switchIndices: map[string]int{},

		seqID: uint16(rng.Int63()),
	}
}

// NewControllerLogin creates a Controller by logging in with a username and
// password.
func NewControllerLogin(email, password string) (*Controller, error) {
	info, err := Login(email, password, "")
	if err != nil {
		return nil, errors.Wrap(err, "new controller")
	}
	return NewController(info, 0), nil
}

// Login creates a new authentication token on the session using the username
// and password.
func (c *Controller) Login(email, password string) error {
	info, err := Login(email, password, "")
	if err != nil {
		return errors.Wrap(err, "login controller")
	}
	c.sessionInfoLock.Lock()
	c.sessionInfo = info
	c.sessionInfoLock.Unlock()
	return nil
}

// Devices enumerates the devices available to the account.
//
// Each device's status is available through its LastStatus() method.
func (c *Controller) Devices() ([]*ControllerDevice, error) {
	sessInfo := c.getSessionInfo()
	devicesResponse, err := GetDevices(sessInfo.UserID, sessInfo.AccessToken)
	if err != nil {
		return nil, err
	}
	var results []*ControllerDevice
	for _, dev := range devicesResponse {
		if !dev.IsOnline && !dev.IsActive {
			// Some devices have no bulbs array, and can cause
			// issues when fetching device properties.
			// https://github.com/unixpickle/cbyge/issues/4
			continue
		}
		props, err := GetDeviceProperties(sessInfo.AccessToken, dev.ProductID, dev.ID)
		if err != nil {
			if !IsPropertyNotExistsError(err) {
				return nil, err
			}
			continue
		}
		for _, bulb := range props.Bulbs {
			cd := &ControllerDevice{
				deviceID: strconv.FormatInt(bulb.DeviceID, 10),
				switchID: bulb.SwitchID,
				name:     bulb.DisplayName,
			}
			results = append(results, cd)
		}
	}
	// Update device status. If this fails, we swallow the error
	// because the device(s) are automatically marked offline.
	c.DeviceStatuses(results)
	return results, nil
}

// DeviceStatus gets the status for a previously enumerated device.
//
// If no error occurs, the status is updated in d.LastStatus() in addition to
// being returned.
func (c *Controller) DeviceStatus(d *ControllerDevice) (ControllerDeviceStatus, error) {
	var packets []*Packet
	seqIDs := map[uint16]bool{}
	c.switchMappingLock.RLock()
	var curSwitch uint32
	if len(c.switches[d.deviceID]) > 0 {
		curSwitch = c.switches[d.deviceID][c.switchIndices[d.deviceID]]
	}
	for _, switchID := range c.switches[d.deviceID] {
		seqID := c.nextSeqID()
		packets = append(packets, NewPacketGetStatusPaginated(switchID, seqID))
		seqIDs[seqID] = true
	}
	c.switchMappingLock.RUnlock()

	if len(packets) == 0 {
		return ControllerDeviceStatus{}, errors.Wrap(UnreachableError, "lookup device status")
	}

	var responsePacket *StatusPaginatedResponse
	var decodeErr error
	var numResponses int
	err := c.callAndWait(packets, false, func(p *Packet) bool {
		if seq, err := p.Seq(); err == nil && p.IsResponse && !seqIDs[seq] {
			// This is a response to a packet we did not send.
			return false
		}
		if IsStatusPaginatedResponse(p) {
			numResponses++
			responses, err := DecodeStatusPaginatedResponse(p)
			if err == nil {
				// Always prioritize a response directly from the actual
				// device, since it will be the most up-to-date.
				switchID := binary.BigEndian.Uint32(p.Data[:4])
				isPrimary := d.isSwitch(switchID)

				for _, resp := range responses {
					if resp.Device == d.deviceIndex() {
						// Prioritize statuses from the device's switch and
						// the switch that we control this device through,
						// since both switches are likely to have the most
						// up-to-date information.
						if responsePacket == nil || switchID == curSwitch || isPrimary {
							// Doing &resp references the for-loop variable.
							responsePacket = new(StatusPaginatedResponse)
							*responsePacket = resp
							if isPrimary {
								return true
							}
						}
					}
				}
			} else {
				decodeErr = err
			}
		} else if p.IsResponse && len(p.Data) >= 4 && p.Data[len(p.Data)-1] != 0 {
			// This is an error response from some switch.
			numResponses++
			if decodeErr == nil {
				decodeErr = RemoteCallError
			}
		}
		return numResponses >= len(packets)
	})

	if responsePacket != nil {
		status := ControllerDeviceStatus{
			StatusPaginatedResponse: *responsePacket,
			IsOnline:                true,
		}
		d.lastStatusLock.Lock()
		d.lastStatus = status
		d.lastStatusLock.Unlock()
		return status, nil
	}

	if decodeErr != nil {
		err = decodeErr
	} else if err == nil {
		err = UnreachableError
	}
	c.switchFailed(d)
	return ControllerDeviceStatus{}, errors.Wrap(err, "lookup device status")
}

// DeviceStatuses gets the status for previously enumerated devices.
//
// Each device will have its own status, and can have an independent error
// when fetching the status.
//
// Each device's status is updated in d.LastStatus() if no error occurred for
// that device.
func (c *Controller) DeviceStatuses(devs []*ControllerDevice) ([]ControllerDeviceStatus, []error) {
	hasResponses := make([]bool, 0, len(devs))
	packets := make([]*Packet, 0, len(devs))
	devIndexToDev := map[int]*ControllerDevice{}
	switchToPacketIndex := map[uint32]int{}
	seqIDs := map[uint16]bool{}
	for _, d := range devs {
		devIndexToDev[d.deviceIndex()] = d
		if d.hasSwitch() {
			switchToPacketIndex[uint32(d.switchID)] = len(packets)
			seqID := c.nextSeqID()
			packet := NewPacketGetStatusPaginated(uint32(d.switchID), seqID)
			packets = append(packets, packet)
			hasResponses = append(hasResponses, false)
			seqIDs[seqID] = true
		}
	}
	if len(packets) == 0 {
		errs := make([]error, len(devs))
		for i := range errs {
			errs[i] = UnreachableError
		}
		return nil, errs
	}

	devToStatus := map[*ControllerDevice]ControllerDeviceStatus{}
	err := c.callAndWait(packets, false, func(p *Packet) bool {
		if seq, err := p.Seq(); err == nil && p.IsResponse && !seqIDs[seq] {
			// This is a response to a packet we did not send.
			return false
		}
		if IsStatusPaginatedResponse(p) {
			switchID := binary.BigEndian.Uint32(p.Data[:4])
			devIdx, ok := switchToPacketIndex[switchID]
			if !ok || hasResponses[devIdx] {
				return false
			}
			hasResponses[devIdx] = true
			responses, err := DecodeStatusPaginatedResponse(p)
			if err == nil {
				for _, resp := range responses {
					dev, ok := devIndexToDev[resp.Device]
					if !ok {
						continue
					}
					devToStatus[dev] = ControllerDeviceStatus{
						IsOnline:                true,
						StatusPaginatedResponse: resp,
					}
					c.addSwitchMapping(dev, switchID)
				}
			}
		} else if p.IsResponse && len(p.Data) >= 4 && p.Data[len(p.Data)-1] != 0 {
			// This is an error response.
			switchID := binary.BigEndian.Uint32(p.Data[:4])
			packetIdx, ok := switchToPacketIndex[switchID]
			if ok && !hasResponses[packetIdx] {
				hasResponses[packetIdx] = true
			}
		}
		for _, hasResponse := range hasResponses {
			if !hasResponse {
				return false
			}
		}
		return true
	})

	// Even if there was no timeout, some devices may simply not
	// be reachable because they aren't connected to any switches.
	if err == nil {
		err = UnreachableError
	}

	// Update statuses for online devices.
	deviceStatuses := make([]ControllerDeviceStatus, len(devs))
	deviceErrors := make([]error, len(devs))
	for i, dev := range devs {
		status, ok := devToStatus[dev]
		if ok {
			devs[i].lastStatusLock.Lock()
			devs[i].lastStatus = status
			devs[i].lastStatusLock.Unlock()
			deviceStatuses[i] = status
		} else {
			deviceErrors[i] = err
		}
	}

	return deviceStatuses, deviceErrors
}

// SetDeviceStatus turns on or off a device.
func (c *Controller) SetDeviceStatus(d *ControllerDevice, status bool) error {
	return c.setDeviceStatus(d, status, false)
}

// SetDeviceStatusAsync is like SetDeviceStatus, but does not wait for the
// device's state to change.
func (c *Controller) SetDeviceStatusAsync(d *ControllerDevice, status bool) error {
	return c.setDeviceStatus(d, status, true)
}

func (c *Controller) setDeviceStatus(d *ControllerDevice, status, async bool) error {
	switchID, err := c.currentSwitch(d)
	if err != nil {
		return errors.Wrap(err, "set device status")
	}
	statusInt := 0
	if status {
		statusInt = 1
	}
	packet := NewPacketSetDeviceStatus(switchID, c.nextSeqID(), d.deviceIndex(), statusInt)
	return c.checkedSwitch(d, c.callAndWaitSimple(packet, "set device status", async))
}

// BlastDeviceStatuses asynchronously turns on or off many devices in bulk.
// It will use up to numSwitches switches per device, providing redundancy if
// some switches are not connected.
// If numSwitches is 0, one switch will be used per device.
func (c *Controller) BlastDeviceStatuses(ds []*ControllerDevice, statuses []bool,
	numSwitches int) error {
	var packets []*Packet
	for i, d := range ds {
		switchIDs, err := c.randomSwitches(d, numSwitches)
		if err != nil {
			return errors.Wrap(err, "blast device statuses")
		}
		statusInt := 0
		if statuses[i] {
			statusInt = 1
		}
		for _, switchID := range switchIDs {
			packet := NewPacketSetDeviceStatus(switchID, c.nextSeqID(), d.deviceIndex(), statusInt)
			packets = append(packets, packet)
		}
	}
	if err := c.blastPackets(packets); err != nil {
		return errors.Wrap(err, "blast device statuses")
	}
	return nil
}

// SetDeviceLum changes a device's brightness.
//
// Brightness values are in [1, 100].
func (c *Controller) SetDeviceLum(d *ControllerDevice, lum int) error {
	return c.setDeviceLum(d, lum, false)
}

// SetDeviceLumAsync is like SetDeviceLum, but does not wait for the device's
// status to change.
func (c *Controller) SetDeviceLumAsync(d *ControllerDevice, lum int) error {
	return c.setDeviceLum(d, lum, true)
}

func (c *Controller) setDeviceLum(d *ControllerDevice, lum int, async bool) error {
	switchID, err := c.currentSwitch(d)
	if err != nil {
		return errors.Wrap(err, "set device luminance")
	}
	packet := NewPacketSetLum(switchID, c.nextSeqID(), d.deviceIndex(), lum)
	return c.checkedSwitch(d, c.callAndWaitSimple(packet, "set device luminance", async))
}

// SetDeviceRGB changes a device's RGB.
func (c *Controller) SetDeviceRGB(d *ControllerDevice, r, g, b uint8) error {
	return c.setDeviceRGB(d, r, g, b, false)
}

// SetDeviceRGBAsync is like SetDeviceRGB, but does not wait for the device's
// status to change.
func (c *Controller) SetDeviceRGBAsync(d *ControllerDevice, r, g, b uint8) error {
	return c.setDeviceRGB(d, r, g, b, true)
}

func (c *Controller) setDeviceRGB(d *ControllerDevice, r, g, b uint8, async bool) error {
	switchID, err := c.currentSwitch(d)
	if err != nil {
		return errors.Wrap(err, "set device RGB")
	}
	packet := NewPacketSetRGB(switchID, c.nextSeqID(), d.deviceIndex(), r, g, b)
	return c.checkedSwitch(d, c.callAndWaitSimple(packet, "set device RGB", async))
}

// SetDeviceCT changes a device's color tone.
//
// Color tone values are in [0, 100].
func (c *Controller) SetDeviceCT(d *ControllerDevice, ct int) error {
	return c.setDeviceCT(d, ct, false)
}

// SetDeviceCTAsync is like SetDeviceCT, but does not wait for the device's
// status to change.
func (c *Controller) SetDeviceCTAsync(d *ControllerDevice, ct int) error {
	return c.setDeviceCT(d, ct, true)
}

func (c *Controller) setDeviceCT(d *ControllerDevice, ct int, async bool) error {
	switchID, err := c.currentSwitch(d)
	if err != nil {
		return errors.Wrap(err, "set device color tone")
	}
	packet := NewPacketSetCT(switchID, c.nextSeqID(), d.deviceIndex(), ct)
	return c.checkedSwitch(d, c.callAndWaitSimple(packet, "set device color tone", async))
}

func (c *Controller) addSwitchMapping(dev *ControllerDevice, switchID uint32) {
	c.switchMappingLock.Lock()
	defer c.switchMappingLock.Unlock()

	// If this is the device's switch, then we should set
	// the device to use this switch since it's known to be
	// accessible.
	updateIndex := dev.isSwitch(switchID)

	for i, x := range c.switches[dev.deviceID] {
		if x == switchID {
			if updateIndex {
				c.switchIndices[dev.deviceID] = i
			}
			return
		}
	}
	c.switches[dev.deviceID] = append(c.switches[dev.deviceID], switchID)
	if updateIndex {
		c.switchIndices[dev.deviceID] = len(c.switches[dev.deviceID]) - 1
	}
}

func (c *Controller) currentSwitch(dev *ControllerDevice) (uint32, error) {
	c.switchMappingLock.RLock()
	defer c.switchMappingLock.RUnlock()

	switches := c.switches[dev.deviceID]
	if len(switches) == 0 {
		return 0, UnreachableError
	}
	return switches[c.switchIndices[dev.deviceID]], nil
}

func (c *Controller) checkedSwitch(dev *ControllerDevice, err error) error {
	if err != nil {
		c.switchFailed(dev)
	}
	return err
}

func (c *Controller) switchFailed(dev *ControllerDevice) {
	c.switchMappingLock.Lock()
	defer c.switchMappingLock.Unlock()
	// Round-robin through supported switches.
	switches := c.switches[dev.deviceID]
	c.switchIndices[dev.deviceID] = (c.switchIndices[dev.deviceID] + 1) % len(switches)
}

func (c *Controller) randomSwitches(dev *ControllerDevice, max int) ([]uint32, error) {
	c.switchMappingLock.RLock()
	defer c.switchMappingLock.RUnlock()
	ordered := c.switches[dev.deviceID]
	if len(ordered) == 0 {
		return nil, UnreachableError
	}
	cur := c.switchIndices[dev.deviceID]
	res := []uint32{ordered[cur]}

	shuffled := append(append([]uint32{}, ordered[:cur]...), ordered[cur+1:]...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return append(res, shuffled[:essentials.MinInt(len(shuffled), max-1)]...), nil
}

func (c *Controller) callAndWaitSimple(p *Packet, context string, async bool) error {
	seq, err := p.Seq()
	if err != nil {
		return err
	}
	// Currently, I have not found a fool-proof way to wait
	// until a status update has completed, aside from polling
	// the device status until the change is visible there.
	//
	// If we use a device's own switch to update the device, then
	// waiting for a response packet seems to be sufficient.
	// Otherwise, the switch may return a response packet before
	// the device's new status is in effect.
	//
	// One thing which seems to work reasonably well is waiting
	// for both a response packet and a sync packet. This doesn't
	// always work, though. Sometimes we can receive an old sync
	// packet from a previous request (for example, if we are
	// changing many lights in a row). Other times, we apparently
	// never receive a sync packet and the call times out.
	gotResponse := false
	gotSync := false
	err = c.callAndWait([]*Packet{p}, true, func(p *Packet) bool {
		seq1, err := p.Seq()
		if err == nil && seq == seq1 && p.IsResponse {
			gotResponse = true
		} else if p.Type == PacketTypeSync {
			gotSync = true
		}
		if async && gotResponse {
			return true
		}
		return gotResponse && gotSync
	})
	if err != nil {
		return errors.Wrap(err, context)
	}
	return nil
}

// callAndWait sends packets on a new PacketConn and waits until f returns
// true on a response, or waits for a timeout.
func (c *Controller) callAndWait(p []*Packet, checkError bool, f func(*Packet) bool) error {
	c.packetConnLock.Lock()
	defer c.packetConnLock.Unlock()

	checkSeqs := map[uint16]bool{}
	for _, packet := range p {
		if seq, err := packet.Seq(); err == nil {
			checkSeqs[seq] = true
		}
	}

	conn, err := NewPacketConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	sessInfo := c.getSessionInfo()
	if err := conn.Auth(sessInfo.UserID, sessInfo.Authorize, c.timeout); err != nil {
		return err
	}

	// Prevent the bg thread from blocking on a
	// channel send forever.
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)

	packets := make(chan *Packet, 16)
	errChan := make(chan error, 1)
	go func() {
		defer close(packets)
		for {
			packet, err := conn.Read()
			if err != nil {
				errChan <- err
				return
			}
			if checkError && packet.IsResponse {
				seq, err := packet.Seq()
				if err == nil && checkSeqs[seq] && len(packet.Data) > 0 {
					if packet.Data[len(packet.Data)-1] != 0 {
						errChan <- RemoteCallError
						return
					}
				}
			}
			select {
			case packets <- packet:
			case <-doneChan:
				return
			}
		}
	}()

	for _, subPacket := range p {
		if err := conn.Write(subPacket); err != nil {
			return err
		}
	}

	timeout := time.After(c.timeout)
	for {
		select {
		case packet, ok := <-packets:
			if !ok {
				// Could be a race condition between packets and errChan.
				select {
				case err := <-errChan:
					return err
				default:
					return errors.New("connection closed")
				}
			}
			if f(packet) {
				return nil
			}
		case err := <-errChan:
			return err
		case <-timeout:
			return errors.New("timeout waiting for response")
		}
	}
}

func (c *Controller) blastPackets(p []*Packet) error {
	c.packetConnLock.Lock()
	defer c.packetConnLock.Unlock()

	checkSeqs := map[uint16]bool{}
	for _, packet := range p {
		if seq, err := packet.Seq(); err == nil {
			checkSeqs[seq] = true
		}
	}

	conn, err := NewPacketConn()
	if err != nil {
		return err
	}

	sessInfo := c.getSessionInfo()
	if err := conn.Auth(sessInfo.UserID, sessInfo.Authorize, c.timeout); err != nil {
		conn.Close()
		return err
	}

	for _, subPacket := range p {
		if err := conn.Write(subPacket); err != nil {
			conn.Close()
			return err
		}
	}

	return conn.Close()
}

func (c *Controller) getSessionInfo() *SessionInfo {
	c.sessionInfoLock.RLock()
	defer c.sessionInfoLock.RUnlock()
	return c.sessionInfo
}

func (c *Controller) nextSeqID() uint16 {
	c.seqIDLock.Lock()
	defer c.seqIDLock.Unlock()
	res := c.seqID
	c.seqID++
	return res
}
