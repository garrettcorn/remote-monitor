//go:build !windows && !linux && !darwin

package telemetry

func collectBattery() *BatteryInfo                 { return nil }
func collectPhysicalDisks() ([]PhysicalDisk, bool) { return nil, false }
func collectDevices() []ConnectedDevice            { return nil }
func collectUsers() []UserSession                  { return nil }
