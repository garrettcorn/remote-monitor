//go:build windows

package audio

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows Core Audio (MMDevice API + IAudioEndpointVolume), called through
// raw COM vtables. Default-device switching uses the long-standing
// (undocumented) IPolicyConfig interface used by the Sound control panel.

var (
	modole32             = windows.NewLazySystemDLL("ole32.dll")
	procCoCreateInstance = modole32.NewProc("CoCreateInstance")
	procPropVariantClear = modole32.NewProc("PropVariantClear")
)

var (
	clsidMMDeviceEnumerator = windows.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	iidIMMDeviceEnumerator  = windows.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	iidIAudioEndpointVolume = windows.GUID{Data1: 0x5CDF2C82, Data2: 0x841E, Data3: 0x4546, Data4: [8]byte{0x97, 0x22, 0x0C, 0xF7, 0x40, 0x78, 0x22, 0x9A}}
	clsidPolicyConfig       = windows.GUID{Data1: 0x870AF99C, Data2: 0x171D, Data3: 0x4F9E, Data4: [8]byte{0xAF, 0x0D, 0xE6, 0x3D, 0xF4, 0x0C, 0x2B, 0xC9}}
	iidIPolicyConfig        = windows.GUID{Data1: 0xF8679F50, Data2: 0x850A, Data3: 0x41CF, Data4: [8]byte{0x9C, 0x72, 0x43, 0x0F, 0x29, 0x02, 0x90, 0xC8}}
	pkeyDeviceFriendlyName  = propertyKey{fmtid: windows.GUID{Data1: 0xA45C254E, Data2: 0xDF1C, Data3: 0x4EFD, Data4: [8]byte{0x80, 0x20, 0x67, 0xD1, 0x46, 0xA8, 0x50, 0xE0}}, pid: 14}
)

const (
	eRender           = 0
	eCapture          = 1
	eConsole          = 0
	deviceStateActive = 1
	clsctxAll         = 0x17
	stgmRead          = 0
	vtLPWSTR          = 31
	rpcEChangedMode   = 0x80010106
)

// Vtable indices.
const (
	enumEnumAudioEndpoints      = 3
	enumGetDefaultAudioEndpoint = 4
	enumGetDevice               = 5
	collGetCount                = 3
	collItem                    = 4
	devActivate                 = 3
	devOpenPropertyStore        = 4
	devGetID                    = 5
	propGetValue                = 5
	volSetMasterScalar          = 7
	volGetMasterScalar          = 9
	volSetMute                  = 14
	volGetMute                  = 15
	volGetStepInfo              = 16
	volStepUp                   = 17
	volStepDown                 = 18
	policySetDefaultEndpoint    = 13
)

type propertyKey struct {
	fmtid windows.GUID
	pid   uint32
}

// propVariant mirrors PROPVARIANT (64-bit layout); only VT_LPWSTR is read.
type propVariant struct {
	vt  uint16
	_   [3]uint16
	str *uint16
	_   uintptr
}

// comIface is a COM interface pointer (memory owned by COM, never moves).
type comIface struct {
	vtbl *[64]uintptr
}

func (o *comIface) call(method int, args ...uintptr) uintptr {
	all := append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)
	r, _, _ := syscall.SyscallN(o.vtbl[method], all...)
	return r
}

func (o *comIface) release() {
	if o != nil {
		o.call(2)
	}
}

// Out-parameter allocators. noinline guarantees heap allocation, so the
// addresses handed to Windows stay valid even if the goroutine stack moves.

//go:noinline
func newIfaceOut() **comIface { return new(*comIface) }

//go:noinline
func newStrOut() **uint16 { return new(*uint16) }

//go:noinline
func newU32Out() *uint32 { return new(uint32) }

//go:noinline
func newF32Out() *float32 { return new(float32) }

//go:noinline
func newPropVariant() *propVariant { return new(propVariant) }

func hresult(r uintptr, what string) error {
	if int32(r) < 0 {
		return fmt.Errorf("%s failed: HRESULT 0x%08X", what, uint32(r))
	}
	return nil
}

// withCOM runs f on a locked OS thread with COM initialised (MTA).
func withCOM(f func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED)
	switch {
	case err == nil, err == syscall.Errno(1): // S_OK, S_FALSE
		defer windows.CoUninitialize()
	case err == syscall.Errno(rpcEChangedMode):
		// Thread already initialised in another mode; usable as is.
	default:
		return fmt.Errorf("CoInitializeEx: %w", err)
	}
	return f()
}

func createInstance(clsid, iid *windows.GUID) (*comIface, error) {
	out := newIfaceOut()
	r, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(clsid)), 0, clsctxAll,
		uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(out)))
	if err := hresult(r, "CoCreateInstance"); err != nil {
		return nil, err
	}
	return *out, nil
}

func deviceID(d *comIface) string {
	out := newStrOut()
	if int32(d.call(devGetID, uintptr(unsafe.Pointer(out)))) < 0 || *out == nil {
		return ""
	}
	id := windows.UTF16PtrToString(*out)
	windows.CoTaskMemFree(unsafe.Pointer(*out))
	return id
}

func friendlyName(d *comIface) string {
	store := newIfaceOut()
	if int32(d.call(devOpenPropertyStore, stgmRead, uintptr(unsafe.Pointer(store)))) < 0 || *store == nil {
		return ""
	}
	defer (*store).release()
	pv := newPropVariant()
	if int32((*store).call(propGetValue, uintptr(unsafe.Pointer(&pkeyDeviceFriendlyName)), uintptr(unsafe.Pointer(pv)))) < 0 {
		return ""
	}
	name := ""
	if pv.vt == vtLPWSTR && pv.str != nil {
		name = windows.UTF16PtrToString(pv.str)
	}
	procPropVariantClear.Call(uintptr(unsafe.Pointer(pv)))
	return name
}

func activateVolume(d *comIface) *comIface {
	out := newIfaceOut()
	if int32(d.call(devActivate, uintptr(unsafe.Pointer(&iidIAudioEndpointVolume)), clsctxAll, 0, uintptr(unsafe.Pointer(out)))) < 0 {
		return nil
	}
	return *out
}

func listFlow(e *comIface, flow uintptr) ([]Device, error) {
	defID := ""
	defOut := newIfaceOut()
	if int32(e.call(enumGetDefaultAudioEndpoint, flow, eConsole, uintptr(unsafe.Pointer(defOut)))) >= 0 && *defOut != nil {
		defID = deviceID(*defOut)
		(*defOut).release()
	}

	collOut := newIfaceOut()
	if err := hresult(e.call(enumEnumAudioEndpoints, flow, deviceStateActive, uintptr(unsafe.Pointer(collOut))), "EnumAudioEndpoints"); err != nil {
		return nil, err
	}
	coll := *collOut
	defer coll.release()

	count := newU32Out()
	coll.call(collGetCount, uintptr(unsafe.Pointer(count)))

	devs := make([]Device, 0, *count)
	for i := uint32(0); i < *count; i++ {
		dOut := newIfaceOut()
		if int32(coll.call(collItem, uintptr(i), uintptr(unsafe.Pointer(dOut)))) < 0 || *dOut == nil {
			continue
		}
		d := *dOut
		dev := Device{ID: deviceID(d), Name: friendlyName(d)}
		dev.Default = dev.ID != "" && dev.ID == defID
		if v := activateVolume(d); v != nil {
			lvl := newF32Out()
			mute := newU32Out()
			v.call(volGetMasterScalar, uintptr(unsafe.Pointer(lvl)))
			v.call(volGetMute, uintptr(unsafe.Pointer(mute)))
			dev.Volume = int(*lvl*100 + 0.5)
			dev.Muted = *mute != 0
			v.release()
		}
		d.release()
		devs = append(devs, dev)
	}
	return devs, nil
}

func get() (*State, error) {
	st := &State{CanSwitch: true}
	err := withCOM(func() error {
		e, err := createInstance(&clsidMMDeviceEnumerator, &iidIMMDeviceEnumerator)
		if err != nil {
			return err
		}
		defer e.release()
		if st.Outputs, err = listFlow(e, eRender); err != nil {
			return err
		}
		st.Inputs, err = listFlow(e, eCapture)
		return err
	})
	if err != nil {
		return nil, err
	}
	return st, nil
}

// withVolume opens the endpoint volume interface of the device with id.
func withVolume(id string, f func(v *comIface) error) error {
	if id == "" {
		return errors.New("device id is required")
	}
	idPtr, err := windows.UTF16PtrFromString(id)
	if err != nil {
		return err
	}
	return withCOM(func() error {
		e, err := createInstance(&clsidMMDeviceEnumerator, &iidIMMDeviceEnumerator)
		if err != nil {
			return err
		}
		defer e.release()
		dOut := newIfaceOut()
		if err := hresult(e.call(enumGetDevice, uintptr(unsafe.Pointer(idPtr)), uintptr(unsafe.Pointer(dOut))), "GetDevice"); err != nil {
			return err
		}
		d := *dOut
		defer d.release()
		v := activateVolume(d)
		if v == nil {
			return errors.New("device has no volume control")
		}
		defer v.release()
		return f(v)
	})
}

func setVolume(id string, level int) error {
	return withVolume(id, func(v *comIface) error {
		return setMasterScalar(v, float32(level)/100)
	})
}

// stepToLevel moves volume to target using VolumeStepUp/Down, for
// architectures where float arguments can't be passed via SyscallN.
func stepToLevel(v *comIface, target float32) error {
	step, count := newU32Out(), newU32Out()
	if err := hresult(v.call(volGetStepInfo, uintptr(unsafe.Pointer(step)), uintptr(unsafe.Pointer(count))), "GetVolumeStepInfo"); err != nil {
		return err
	}
	if *count < 2 {
		return errors.New("device reports no volume steps")
	}
	want := int(target*float32(*count-1) + 0.5)
	for i := 0; i < int(*count) && int(*step) != want; i++ {
		method := volStepUp
		if int(*step) > want {
			method = volStepDown
		}
		if err := hresult(v.call(method, 0), "VolumeStep"); err != nil {
			return err
		}
		v.call(volGetStepInfo, uintptr(unsafe.Pointer(step)), uintptr(unsafe.Pointer(count)))
	}
	return nil
}

func setMute(id string, muted bool) error {
	return withVolume(id, func(v *comIface) error {
		var b uintptr
		if muted {
			b = 1
		}
		return hresult(v.call(volSetMute, b, 0), "SetMute")
	})
}

func setDefault(id string) error {
	if id == "" {
		return errors.New("device id is required")
	}
	idPtr, err := windows.UTF16PtrFromString(id)
	if err != nil {
		return err
	}
	return withCOM(func() error {
		pc, err := createInstance(&clsidPolicyConfig, &iidIPolicyConfig)
		if err != nil {
			return err
		}
		defer pc.release()
		for role := uintptr(0); role <= 2; role++ { // console, multimedia, communications
			if err := hresult(pc.call(policySetDefaultEndpoint, uintptr(unsafe.Pointer(idPtr)), role), "SetDefaultEndpoint"); err != nil {
				return err
			}
		}
		return nil
	})
}
