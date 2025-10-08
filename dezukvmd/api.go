package main

import "net/http"

func registerAPIRoutes() {
	// Start the web server
	http.Handle("/", http.FileServer(webfs))
	http.HandleFunc("/hid", usbKVM.HIDWebSocketHandler)
	http.HandleFunc("/audio", usbCaptureDevice.AudioStreamingHandler)
	http.HandleFunc("/stream", usbCaptureDevice.ServeVideoStream)
}

// Aux APIs for USB KVM mode
func registerLocalAuxRoutes() {
	http.HandleFunc("/aux/switchusbkvm", auxMCU.HandleSwitchUSBToKVM)
	http.HandleFunc("/aux/switchusbremote", auxMCU.HandleSwitchUSBToRemote)
	http.HandleFunc("/aux/presspower", auxMCU.HandlePressPowerButton)
	http.HandleFunc("/aux/releasepower", auxMCU.HandleReleasePowerButton)
	http.HandleFunc("/aux/pressreset", auxMCU.HandlePressResetButton)
	http.HandleFunc("/aux/releasereset", auxMCU.HandleReleaseResetButton)
	http.HandleFunc("/aux/getuuid", auxMCU.HandleGetUUID)
}

// Dummy Aux APIs for setups that do not have an aux MCU
func registerDummyLocalAuxRoutes() {
	dummyHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("Not implemented"))
	}

	http.HandleFunc("/aux/switchusbkvm", dummyHandler)
	http.HandleFunc("/aux/switchusbremote", dummyHandler)
	http.HandleFunc("/aux/presspower", dummyHandler)
	http.HandleFunc("/aux/releasepower", dummyHandler)
	http.HandleFunc("/aux/pressreset", dummyHandler)
	http.HandleFunc("/aux/releasereset", dummyHandler)
	http.HandleFunc("/aux/getuuid", dummyHandler)
}
