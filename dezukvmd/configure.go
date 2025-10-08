package main

import (
	"log"
	"time"

	"imuslab.com/dezukvm/dezukvmd/mod/kvmhid"
)

func SetupHIDCommunication(config *UsbKvmConfig) error {
	// Initiate the HID controller
	usbKVM = kvmhid.NewHIDController(&kvmhid.Config{
		PortName:          config.USBKVMDevicePath,
		BaudRate:          config.USBKVMBaudrate,
		ScrollSensitivity: 0x01, // Set mouse scroll sensitivity
	})

	//Start the HID controller
	err := usbKVM.Connect()
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(1 * time.Second) // Wait for the controller to initialize
	log.Println("Updating chip baudrate to 115200...")
	//Configure the HID controller
	err = usbKVM.ConfigureChipTo115200()
	if err != nil {
		log.Fatalf("Failed to configure chip baudrate: %v", err)
		return err
	}
	time.Sleep(1 * time.Second)

	log.Println("Setting chip USB device properties...")
	time.Sleep(2 * time.Second) // Wait for the controller to initialize
	_, err = usbKVM.WriteChipProperties()
	if err != nil {
		log.Fatalf("Failed to write chip properties: %v", err)
		return err
	}

	log.Println("Configuration command sent. Unplug the device and plug it back in to apply the changes.")
	return nil
}
