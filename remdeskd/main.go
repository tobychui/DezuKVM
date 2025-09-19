package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"imuslab.com/remdeskvm/remdeskd/mod/remdeshid"
	"imuslab.com/remdeskvm/remdeskd/mod/usbcapture"
)

const defaultDevMode = true

var (
	developent        = flag.Bool("dev", defaultDevMode, "Enable development mode with local static files")
	mode              = flag.String("mode", "usbkvm", "Mode of operation: kvm or capture")
	usbKVMDeviceName  = flag.String("usbkvm", "/dev/ttyACM0", "USB KVM device file path")
	usbKVMBaudRate    = flag.Int("baudrate", 115200, "USB KVM baud rate")
	captureDeviceName = flag.String("capture", "/dev/video0", "Video capture device file path")
	usbKVM            *remdeshid.Controller
	videoCapture      *usbcapture.Instance
)

/* Web Server Static Files */
//go:embed www
var embeddedFiles embed.FS
var webfs http.FileSystem

func init() {
	// Initiate the web server static files
	if *developent {
		webfs = http.Dir("./www")
	} else {
		// Embed the ./www folder and trim the prefix
		subFS, err := fs.Sub(embeddedFiles, "www")
		if err != nil {
			log.Fatal(err)
		}
		webfs = http.FS(subFS)
	}
}

func main() {
	flag.Parse()

	// Initiate the HID controller
	usbKVM = remdeshid.NewHIDController(&remdeshid.Config{
		PortName:          *usbKVMDeviceName,
		BaudRate:          *usbKVMBaudRate,
		ScrollSensitivity: 0x01, // Set mouse scroll sensitivity
	})

	switch *mode {
	case "cfgchip":
		//Start the HID controller
		err := usbKVM.Connect()
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(2 * time.Second) // Wait for the controller to initialize
		//Configure the HID controller
		usbKVM.ConfigureChipTo19200()
	case "usbkvm":
		log.Println("Starting in USB KVM mode...")

		//Start the HID controller
		err := usbKVM.Connect()
		if err != nil {
			log.Fatal(err)
		}

		// Initiate the video capture device
		videoCapture, err = usbcapture.NewInstance(&usbcapture.Config{
			DeviceName: *captureDeviceName,
		})

		if err != nil {
			log.Fatalf("Failed to create video capture instance: %v", err)
		}

		//Get device information for debug
		usbcapture.PrintV4L2FormatInfo(*captureDeviceName)

		//Start the video capture device
		err = videoCapture.StartVideoCapture(&usbcapture.CaptureResolution{
			Width:  1920,
			Height: 1080,
			FPS:    10,
		})
		if err != nil {
			log.Fatal(err)
		}

		// Handle program exit to close the HID controller

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			log.Println("Shutting down usbKVM...")
			//usbKVM.Close() //To fix close stuck layer
			log.Println("Shutting down capture device...")
			videoCapture.Close()
			os.Exit(0)
		}()

		// Start the web server
		http.Handle("/", http.FileServer(webfs))
		http.HandleFunc("/hid", usbKVM.HIDWebSocketHandler)
		http.HandleFunc("/stream", videoCapture.ServeVideoStream)
		addr := ":9000"
		log.Printf("Serving on http://localhost%s\n", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	default:
		log.Fatalf("Unknown mode: %s. Supported modes are: usbkvm, capture", *mode)
	}
}
