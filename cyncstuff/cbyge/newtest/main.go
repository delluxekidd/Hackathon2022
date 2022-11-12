package main

import (
	"fmt"

	"github.com/unixpickle/cbyge"
)

func getDeviceInfo(sessionInfo *cbyge.SessionInfo) (int, int) {
	// Create new controller with timeout time duartion
	ctrl := cbyge.NewController(sessionInfo, 10)

	// Get all devices
	devs, _ := ctrl.Devices()

	// Loop through devices and get their status
	var onDevices int
	onDevices = 0
	var colorSum int
	colorSum = 0
	for _, dev := range devs {
		status := dev.LastStatus()
		// Get last element of status {a, b}
		if status.IsOnline {
			// Check if device is on
			if status.StatusPaginatedResponse.IsOn {
				onDevices++
				colorSum += int(status.StatusPaginatedResponse.ColorTone)
			}
		}
	}

	if onDevices == 0 {
		return 0, 0
	}

	return onDevices, colorSum / onDevices
}

func main() {
	callback, _ := cbyge.Login2FA("luke5083@live.com", "Lukeluke44", "")

	// Read input from user
	// ...
	var code string
	fmt.Scan(&code)

	// OTP Callback
	sessionInfo, _ := callback(code)

	fmt.Println(getDeviceInfo(sessionInfo))
}
