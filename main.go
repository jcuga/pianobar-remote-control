package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {

	usernamePtr := flag.String("u", "", "Username")
	passPtr := flag.String("p", "", "Password")
	stationPtr := flag.Uint("s", 0, "Default station number on start.")
	maxStationPtr := flag.Uint("m", 10, "Max number of stations/how high next-channel goes before wraps back to 0.")
	hardBanPtr := flag.Bool("hardban", false, "If set, the 'ban' button does a perm ban instead of 1 month. (issues ban instead of tired command).")
	listenPtr := flag.String("http", "0.0.0.0:7890", "Listen address for serving web remote control.")
	flag.Parse()

	if len(*usernamePtr) == 0 {
		fmt.Println("Must provide username (-u)")
		os.Exit(1)
	}

	if len(*passPtr) == 0 {
		fmt.Println("Must provide password (-p)")
		os.Exit(1)
	}

	if *stationPtr > *maxStationPtr {
		fmt.Printf("default station (-s) cannot be greater than max station (-m).")
		os.Exit(1)
	}
	cmd := exec.Command("pianobar")

	stdin, err := cmd.StdinPipe()

	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	stdin.Write([]byte(*usernamePtr))
	stdin.Write([]byte("\n"))
	stdin.Write([]byte(*passPtr))
	stdin.Write([]byte("\n"))
	stdin.Write([]byte(strconv.FormatUint(uint64(*stationPtr), 10)))
	stdin.Write([]byte("\n"))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	commands := make(chan string, 1)

	http.HandleFunc("/", getWebHandler(commands, *stationPtr, *maxStationPtr, *hardBanPtr))
	log.Printf("Listening on http://%s, playing station: %d of %d. Hard-ban: %t", *listenPtr, *stationPtr, *maxStationPtr, *hardBanPtr)
	go func() {
		log.Fatal(http.ListenAndServe(*listenPtr, nil))
	}()

Loop:
	for {
		select {
		case cmd := <-commands:
			stdin.Write([]byte(cmd))
		case <-done:
			fmt.Println("Exiting because signal.")
			stdin.Write([]byte("\n\nq\n"))
			break Loop
		}
	}
}

func getWebHandler(commands chan<- string, defaultStation uint, maxStation uint, hardBan bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		command := r.URL.Query().Get("cmd")
		station := r.URL.Query().Get("s")
		curStation := defaultStation
		if len(station) > 0 {
			if x, err := strconv.ParseUint(station, 10, 32); err == nil {
				curStation = uint(x)
			} else {
				log.Printf("Error parsing station %q: %v, using default: %d", station, err, defaultStation)
				curStation = uint(defaultStation)
			}
		}

		prevStation := curStation - 1
		// NOTE: can't check prevStation < 0 as uint! Check curStation == 0 instead
		if curStation == 0 {
			prevStation = maxStation
		}
		nextStation := curStation + 1
		if nextStation > maxStation {
			nextStation = 0
		}

		if len(command) > 0 {
			log.Printf("Got command: %q, current station: %d, prevStation: %d, nextStation: %d", command, curStation, prevStation, nextStation)
			switch command {
			case "pause":
				commands <- " "
			case "next":
				commands <- "n"
			case "fav":
				commands <- "+"
			case "ban":
				if hardBan {
					commands <- "-"
				} else {
					// Use "tired" to ban for 1 month instead of perm ban
					// in case accidentally clicked wrong button.
					commands <- "t"
				}
			case "volup":
				commands <- ")"
			case "volupmore":
				commands <- "))))"
			case "voldown":
				commands <- "("
			case "voldownmore":
				commands <- "(((("
			case "station":
				commands <- fmt.Sprintf("s%d\n", curStation)
			default:
				log.Printf("Unrecognized command: %q", command)
			}
		}

		fmt.Fprintf(w, `
		<html>
		<head>
			<title>pianobar-rc (%d)</title>
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				body {
					margin: 0;
					padding: 0;
					background-color: 0;
					max-width: 650px;
					margin-left: auto;
					margin-right: auto;
				}

				a {
					font-size: 2em;
					text-decoration: none;
					color: black;
					display: inline-block;
					padding: 0 0 0 0;
					margin: 0.25%%;
					font-weight: bold;
					width: 49%%;
					height: 20%%;
					text-align: center;
					vertical-align: middle;
				}
			</style>
		</head>
		<body><a href="/?cmd=pause&s=%d" style="background-color: yellow;">(un)pause</a><a href="/?cmd=next&s=%d" style="background-color: orange;">next</a><a href="/?cmd=fav&s=%d" style="background-color: pink;">fav</a><a href="/?cmd=ban&s=%d" style="background-color: red;">ban</a><a href="/?cmd=volup&s=%d" style="background-color: green;">vol +</a><a href="/?cmd=volupmore&s=%d" style="background-color: lightgreen;">vol ++</a><a href="/?cmd=voldown&s=%d" style="background-color: lightblue;">vol -</a><a href="/?cmd=voldownmore&s=%d" style="background-color: blue;">vol --</a><a href="/?cmd=station&s=%d" style="background-color: chocolate;">station - (%d)</a><a href="/?cmd=station&s=%d" style="background-color: magenta;">station + (%d)</a></body>
		</html>`,
			curStation, curStation, curStation, curStation,
			curStation, curStation, curStation, curStation,
			curStation, prevStation, prevStation, nextStation, nextStation)
	}
}
