package main

//go:generate go-bindata -pkg main -o bindata.go -modtime 1 -prefix html html

// build: GOOS=windows GOARCH=amd64 go build -o fskneeboard.exe vfrmap-for-vr/vfrmap

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"vfrmap-for-vr/_vendor/premium/charts"
	"vfrmap-for-vr/_vendor/premium/drm"
	"vfrmap-for-vr/simconnect"
	"vfrmap-for-vr/vfrmap/html/leafletjs"
	"vfrmap-for-vr/vfrmap/html/premium"
	"vfrmap-for-vr/vfrmap/websockets"
)

type Report struct {
	simconnect.RecvSimobjectDataByType
	Title         [256]byte `name:"TITLE"`
	Altitude      float64   `name:"INDICATED ALTITUDE" unit:"feet"` // PLANE ALTITUDE or PLANE ALT ABOVE GROUND
	Latitude      float64   `name:"PLANE LATITUDE" unit:"degrees"`
	Longitude     float64   `name:"PLANE LONGITUDE" unit:"degrees"`
	Heading       float64   `name:"PLANE HEADING DEGREES TRUE" unit:"degrees"`
	Airspeed      float64   `name:"AIRSPEED INDICATED" unit:"knot"`
	AirspeedTrue  float64   `name:"AIRSPEED TRUE" unit:"knot"`
	VerticalSpeed float64   `name:"VERTICAL SPEED" unit:"ft/min"`
	Flaps         float64   `name:"TRAILING EDGE FLAPS LEFT ANGLE" unit:"degrees"`
	Trim          float64   `name:"ELEVATOR TRIM PCT" unit:"percent"`
	RudderTrim    float64   `name:"RUDDER TRIM PCT" unit:"percent"`
}

func (r *Report) RequestData(s *simconnect.SimConnect) {
	defineID := s.GetDefineID(r)
	requestID := defineID
	s.RequestDataOnSimObjectType(requestID, defineID, 0, simconnect.SIMOBJECT_TYPE_USER)
}

type TrafficReport struct {
	simconnect.RecvSimobjectDataByType
	AtcID           [64]byte `name:"ATC ID"`
	AtcFlightNumber [8]byte  `name:"ATC FLIGHT NUMBER"`
	Altitude        float64  `name:"PLANE ALTITUDE" unit:"feet"`
	Latitude        float64  `name:"PLANE LATITUDE" unit:"degrees"`
	Longitude       float64  `name:"PLANE LONGITUDE" unit:"degrees"`
	Heading         float64  `name:"PLANE HEADING DEGREES TRUE" unit:"degrees"`
}

func (r *TrafficReport) RequestData(s *simconnect.SimConnect) {
	defineID := s.GetDefineID(r)
	requestID := defineID
	s.RequestDataOnSimObjectType(requestID, defineID, 0, simconnect.SIMOBJECT_TYPE_AIRCRAFT)
}

func (r *TrafficReport) Inspect() string {
	return fmt.Sprintf(
		"%s GPS %.6f %.6f @ %.0f feet %.0f°",
		r.AtcID,
		r.Latitude,
		r.Longitude,
		r.Altitude,
		r.Heading,
	)
}

type TeleportRequest struct {
	simconnect.RecvSimobjectDataByType
	Latitude  float64 `name:"PLANE LATITUDE" unit:"degrees"`
	Longitude float64 `name:"PLANE LONGITUDE" unit:"degrees"`
	Altitude  float64 `name:"PLANE ALTITUDE" unit:"feet"`
}

func (r *TeleportRequest) SetData(s *simconnect.SimConnect) {
	defineID := s.GetDefineID(r)

	buf := [3]float64{
		r.Latitude,
		r.Longitude,
		r.Altitude,
	}

	size := simconnect.DWORD(3 * 8) // 2 * 8 bytes
	s.SetDataOnSimObject(defineID, simconnect.OBJECT_ID_USER, 0, 0, size, unsafe.Pointer(&buf[0]))
}

func shutdownWithPromt() {
	buf := bufio.NewReader(os.Stdin)
	fmt.Print("\nPress ENTER to continue...")
	buf.ReadBytes('\n')

	os.Exit(0)
}

var buildVersion string
var buildTime string
var pro string

var bPro bool
var productName string

var disableTeleport bool
var devMode bool

var verbose bool
var httpListen string

func main() {
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.StringVar(&httpListen, "listen", "0.0.0.0:9000", "http listen")
	flag.BoolVar(&disableTeleport, "disable-teleport", false, "disable teleport")
	flag.BoolVar(&devMode, "dev", false, "enable dev mode, i.e. no running msfs required")
	flag.Parse()

	bPro = pro == "true"

	productName = "FSKneeboard"
	if bPro {
		productName += " PRO"
	}

	fmt.Printf("\n"+productName+" - Server\n  Website: https://fskneeboard.com\n  Readme:  https://github.com/Christian1984/vfrmap-for-vr/blob/master/README.md\n  Issues:  https://github.com/Christian1984/vfrmap-for-vr/issues\n  Version: %s (%s)\n\n", buildVersion, buildTime)

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM)
	exePath, _ := os.Executable()

	if bPro {
		if !drm.Valid() {
			fmt.Println("\nWARNING: You do not have a valid license to run FSKneeboard PRO!")
			fmt.Println("Please purchase a license at https://fskneeboard.com/buy-now and place your fskneeboard.lic-file in the same directory as fskneeboard-server.exe.")
			shutdownWithPromt()
		} else {
			fmt.Println("Valid license found!")
			fmt.Println("Thanks for purchasing FSKneeboard PRO and supporting the development of this mod!")
			fmt.Println("")
		}
	} else {
		fmt.Println("Thanks for trying FSKneeboard FREE!")
		fmt.Println("Please checkout https://fskneeboard.com and purchase FSKneeboard PRO to unlock all features the extension has to offer.")
		fmt.Println("")
	}

	ws := websockets.New()

	s, err := simconnect.New(productName)
	if err != nil {
		fmt.Println("Flight Simulator not running!")

		if !devMode {
			fmt.Println("Run with option -dev for development purposes without a Flight Simulator connection...")
			shutdownWithPromt()
		}
	} else {
		fmt.Println("Connected to Flight Simulator!")
	}

	report := &Report{}
	trafficReport := &TrafficReport{}
	teleportReport := &TeleportRequest{}

	eventSimStartID := simconnect.DWORD(0)
	startupTextEventID := simconnect.DWORD(0)

	if s != nil {
		err = s.RegisterDataDefinition(report)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		err = s.RegisterDataDefinition(trafficReport)
		if err != nil {
			panic(err)
		}

		err = s.RegisterDataDefinition(teleportReport)
		if err != nil {
			panic(err)
		}

		eventSimStartID = s.GetEventID()
		//s.SubscribeToSystemEvent(eventSimStartID, "SimStart")
		//s.SubscribeToFacilities(simconnect.FACILITY_LIST_TYPE_AIRPORT, s.GetDefineID(&simconnect.DataFacilityAirport{}))
		//s.SubscribeToFacilities(simconnect.FACILITY_LIST_TYPE_WAYPOINT, s.GetDefineID(&simconnect.DataFacilityWaypoint{}))

		startupTextEventID = s.GetEventID()
		s.ShowText(simconnect.TEXT_TYPE_PRINT_WHITE, 15, startupTextEventID, ">> FSKneeboard connected <<")
	}

	go func() {
		charts.UpdateIndex()

		setHeaders := func(contentType string, w http.ResponseWriter) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			//w.Header().Set("Content-Type", contentType)
		}

		sendResponse := func(contentType string, w http.ResponseWriter, r *http.Request, filePath string, requestedResource string, asset []byte) {
			setHeaders(contentType, w)

			if _, err = os.Stat(filePath); os.IsNotExist(err) {
				fmt.Println("use embedded", requestedResource)
				w.Write(asset)
			} else {
				fmt.Println("use local", filePath)
				http.ServeFile(w, r, filePath)
			}
		}

		vfrmap := func(w http.ResponseWriter, r *http.Request) {
			filePath := filepath.Join(filepath.Dir(exePath), "vfrmap", "html", "index.html")
			sendResponse("text/html", w, r, filePath, "index.html", MustAsset(filepath.Base(filePath)))
		}

		premium := func(w http.ResponseWriter, r *http.Request) {
			requestedResource := strings.TrimPrefix(r.URL.Path, "/premium/")
			fmt.Println("requestedResource", requestedResource)
			filePath := filepath.Join(filepath.Dir(exePath), "_vendor", "premium", requestedResource)
			sendResponse("text/html", w, r, filePath, requestedResource, premium.MustAsset(requestedResource))
		}

		chartsIndex := func(w http.ResponseWriter, r *http.Request) {
			requestedResource := strings.TrimPrefix(r.URL.Path, "/premium/chartsIndex/")
			fmt.Println("requestedResource", requestedResource)
			setHeaders("application/json", w)
			charts.Json(w, r)
		}

		chartServer := http.FileServer(http.Dir("./charts"))

		http.HandleFunc("/ws", ws.Serve)
		http.HandleFunc("/premium/", premium)
		http.HandleFunc("/premium/chartsIndex", chartsIndex)
		http.Handle("/leafletjs/", http.StripPrefix("/leafletjs/", leafletjs.FS{}))
		http.Handle("/premium/charts/", http.StripPrefix("/premium/charts/", chartServer))
		http.HandleFunc("/", vfrmap)

		err := http.ListenAndServe(httpListen, nil)
		if err != nil {
			panic(err)
		}
	}()

	//autosaveTick := time.NewTicker(5 * time.Minute)
	simconnectTick := time.NewTicker(100 * time.Millisecond)
	planePositionTick := time.NewTicker(200 * time.Millisecond)
	trafficPositionTick := time.NewTicker(10000 * time.Millisecond)

	for {
		select {
		/*case <-autosaveTick.C:
		  if s == nil {
		      continue
		  }

		  pwd, err := os.Getwd()
		  if err == nil {
		      t := time.Now()
		      ts := t.Format("2006-01-02T15-04-05")
		      fn := pwd + "\\autosave\\" + ts + ".FLT"
		      fmt.Println("Creating Autosave as " + fn)
		      s.FlightSave(fn, "test", "test");
		  }
		*/

		case <-planePositionTick.C:
			if s == nil {
				continue
			}

			report.RequestData(s)

		case <-trafficPositionTick.C:
			if s == nil {
				continue
			}

		case <-simconnectTick.C:
			if s == nil {
				continue
			}

			ppData, r1, err := s.GetNextDispatch()

			if r1 < 0 {
				if uint32(r1) == simconnect.E_FAIL {
					// skip error, means no new messages?
					continue
				} else {
					panic(fmt.Errorf("GetNextDispatch error: %d %s", r1, err))
				}
			}

			recvInfo := *(*simconnect.Recv)(ppData)

			switch recvInfo.ID {
			case simconnect.RECV_ID_EXCEPTION:
				recvErr := *(*simconnect.RecvException)(ppData)
				fmt.Printf("SIMCONNECT_RECV_ID_EXCEPTION %#v\n", recvErr)

			case simconnect.RECV_ID_OPEN:
				recvOpen := *(*simconnect.RecvOpen)(ppData)
				fmt.Printf(
					"\nFlight Simulator Info:\n  Codename: %s\n  Version: %d.%d (%d.%d)\n  Simconnect: %d.%d (%d.%d)\n\n",
					recvOpen.ApplicationName,
					recvOpen.ApplicationVersionMajor,
					recvOpen.ApplicationVersionMinor,
					recvOpen.ApplicationBuildMajor,
					recvOpen.ApplicationBuildMinor,
					recvOpen.SimConnectVersionMajor,
					recvOpen.SimConnectVersionMinor,
					recvOpen.SimConnectBuildMajor,
					recvOpen.SimConnectBuildMinor,
				)
				fmt.Printf("Ready... Please leave this window open during your Flight Simulator session. Have a safe flight :-)\n\n")

			case simconnect.RECV_ID_EVENT:
				recvEvent := *(*simconnect.RecvEvent)(ppData)

				switch recvEvent.EventID {
				case eventSimStartID:
					fmt.Println("EVENT: SimStart")
				case startupTextEventID:
					// ignore
				default:
					fmt.Println("unknown SIMCONNECT_RECV_ID_EVENT", recvEvent.EventID)
				}
			case simconnect.RECV_ID_WAYPOINT_LIST:
				waypointList := *(*simconnect.RecvFacilityWaypointList)(ppData)
				fmt.Printf("SIMCONNECT_RECV_ID_WAYPOINT_LIST %#v\n", waypointList)

			case simconnect.RECV_ID_AIRPORT_LIST:
				airportList := *(*simconnect.RecvFacilityAirportList)(ppData)
				fmt.Printf("SIMCONNECT_RECV_ID_AIRPORT_LIST %#v\n", airportList)

			case simconnect.RECV_ID_SIMOBJECT_DATA_BYTYPE:
				recvData := *(*simconnect.RecvSimobjectDataByType)(ppData)

				switch recvData.RequestID {
				case s.DefineMap["Report"]:
					report = (*Report)(ppData)

					if verbose {
						fmt.Printf("REPORT: %#v\n", report)
					}

					ws.Broadcast(map[string]interface{}{
						"type":           "plane",
						"latitude":       report.Latitude,
						"longitude":      report.Longitude,
						"altitude":       fmt.Sprintf("%.0f", report.Altitude),
						"heading":        int(report.Heading),
						"airspeed":       fmt.Sprintf("%.0f", report.Airspeed),
						"airspeed_true":  fmt.Sprintf("%.0f", report.AirspeedTrue),
						"vertical_speed": fmt.Sprintf("%.0f", report.VerticalSpeed),
						"flaps":          fmt.Sprintf("%.0f", report.Flaps),
						"trim":           fmt.Sprintf("%.1f", report.Trim),
						"rudder_trim":    fmt.Sprintf("%.1f", report.RudderTrim),
					})

				case s.DefineMap["TrafficReport"]:
					trafficReport = (*TrafficReport)(ppData)
					fmt.Printf("TRAFFIC REPORT: %s\n", trafficReport.Inspect())
				}

			case simconnect.RECV_ID_QUIT:
				fmt.Println("Flight Simulator was shut down. Exiting...")
				shutdownWithPromt()

			default:
				fmt.Println("recvInfo.ID unknown", recvInfo.ID)
			}

		case <-exitSignal:
			fmt.Println("Exiting...")
			if s != nil {
				s.Close()
				if err != nil {
					panic(err)
				}
			}
			os.Exit(0)

		case _ = <-ws.NewConnection:
			// drain and skip

		case m := <-ws.ReceiveMessages:
			fmt.Println("ws.ReceiveMessages!")
			if s == nil {
				continue
			}
			handleClientMessage(m, s)
		}
	}
}

func handleClientMessage(m websockets.ReceiveMessage, s *simconnect.SimConnect) {
	var pkt map[string]interface{}
	if err := json.Unmarshal(m.Message, &pkt); err != nil {
		fmt.Println("invalid websocket packet", err)
	} else {
		pktType, ok := pkt["type"].(string)
		if !ok {
			fmt.Println("invalid websocket packet", pkt)
			return
		}
		switch pktType {
		case "teleport":
			if disableTeleport {
				fmt.Println("teleport disabled", pkt)
				return
			}

			// validate user input
			lat, ok := pkt["lat"].(float64)
			if !ok {
				fmt.Println("invalid websocket packet", pkt)
				return
			}
			lng, ok := pkt["lng"].(float64)
			if !ok {
				fmt.Println("invalid websocket packet", pkt)
				return
			}
			altitude, ok := pkt["altitude"].(float64)
			if !ok {
				fmt.Println("invalid websocket packet", pkt)
				return
			}

			// teleport
			r := &TeleportRequest{Latitude: lat, Longitude: lng, Altitude: altitude}
			r.SetData(s)
		}
	}
}