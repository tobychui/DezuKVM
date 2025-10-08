package main

/*
	usbkvm.go

	Handles the USB KVM device connections and auxiliary devices
	running in USB KVM mode. This mode only support 1 USB KVM device
	at a time.

	For running multiple USB KVM devices, use the ipkvm mode.
*/
import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"imuslab.com/dezukvm/dezukvmd/mod/kvmaux"
	"imuslab.com/dezukvm/dezukvmd/mod/kvmhid"
	"imuslab.com/dezukvm/dezukvmd/mod/usbcapture"
)

type UsbKvmConfig struct {
	ListeningAddress        string
	USBKVMDevicePath        string
	AuxMCUDevicePath        string
	VideoCaptureDevicePath  string
	AudioCaptureDevicePath  string
	CaptureResolutionWidth  int
	CaptureResolutionHeight int
	CaptureResolutionFPS    int
	USBKVMBaudrate          int
	AuxMCUBaudrate          int
}

var (
	/* Internal variables for USB-KVM mode only */
	usbKVM              *kvmhid.Controller
	auxMCU              *kvmaux.AuxMcu
	usbCaptureDevice    *usbcapture.Instance
	defaultUsbKvmConfig = &UsbKvmConfig{
		ListeningAddress:        ":9000",
		USBKVMDevicePath:        "/dev/ttyUSB0",
		AuxMCUDevicePath:        "/dev/ttyACM0",
		VideoCaptureDevicePath:  "/dev/video0",
		AudioCaptureDevicePath:  "/dev/snd/pcmC1D0c",
		CaptureResolutionWidth:  1920,
		CaptureResolutionHeight: 1080,
		CaptureResolutionFPS:    25,
		USBKVMBaudrate:          115200,
		AuxMCUBaudrate:          115200,
	}
)

func loadUsbKvmConfig() (*UsbKvmConfig, error) {
	if _, err := os.Stat(usbKvmConfigPath); os.IsNotExist(err) {
		file, err := os.OpenFile(usbKvmConfigPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return nil, err
		}

		// Save default config as JSON
		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultUsbKvmConfig); err != nil {
			file.Close()
			return nil, err
		}
		file.Close()
		return defaultUsbKvmConfig, nil
	}

	// Load config from file
	file, err := os.Open(usbKvmConfigPath)
	if err != nil {
		return nil, err
	}

	cfg := &UsbKvmConfig{}
	dec := json.NewDecoder(file)
	if err := dec.Decode(cfg); err != nil {
		file.Close()
		return nil, err
	}
	file.Close()
	return cfg, nil
}

func startUsbKvmMode(config *UsbKvmConfig) error {
	log.Println("Starting in USB KVM mode...")
	// Initiate the HID controller
	usbKVM = kvmhid.NewHIDController(&kvmhid.Config{
		PortName:          config.USBKVMDevicePath,
		BaudRate:          config.USBKVMBaudrate,
		ScrollSensitivity: 0x01, // Set mouse scroll sensitivity
	})

	//Start the HID controller
	err := usbKVM.Connect()
	if err != nil {
		return err
	}

	//Start auxiliary MCU connections
	auxMCU, err = kvmaux.NewAuxOutbandController(config.AuxMCUDevicePath, config.AuxMCUBaudrate)
	if err != nil {
		return err
	}

	//Try get the UUID from the auxiliary MCU
	uuid, err := auxMCU.GetUUID()
	if err != nil {
		log.Println("Get UUID failed:", err, " - Auxiliary MCU may not be connected.")

		//Register dummy AUX routes if failed to get UUID
		registerDummyLocalAuxRoutes()
	} else {
		log.Println("Auxiliary MCU found with UUID:", uuid)

		//Register the AUX routes if success
		registerLocalAuxRoutes()
	}

	// Initiate the video capture device
	usbCaptureDevice, err = usbcapture.NewInstance(&usbcapture.Config{
		VideoDeviceName: config.VideoCaptureDevicePath,
		AudioDeviceName: config.AudioCaptureDevicePath,
		AudioConfig:     usbcapture.GetDefaultAudioConfig(),
	})

	if err != nil {
		log.Println("Video capture device init failed:", err, " - Video capture device may not be connected.")
		return err
	}

	//Get device information for debug
	usbcapture.PrintV4L2FormatInfo(config.VideoCaptureDevicePath)

	//Start the video capture device
	err = usbCaptureDevice.StartVideoCapture(&usbcapture.CaptureResolution{
		Width:  config.CaptureResolutionWidth,
		Height: config.CaptureResolutionHeight,
		FPS:    config.CaptureResolutionFPS,
	})
	if err != nil {
		return err
	}

	// Handle program exit to close the HID controller
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutting down usbKVM...")

		if auxMCU != nil {
			auxMCU.Close()
		}
		log.Println("Shutting down capture device...")
		if usbCaptureDevice != nil {
			usbCaptureDevice.Close()
		}
		os.Exit(0)
	}()

	// Register the rest of the API routes
	registerAPIRoutes()

	addr := config.ListeningAddress
	log.Printf("Serving on%s\n", addr)
	err = http.ListenAndServe(addr, nil)
	return err
}
