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

	usernamePtr := flag.String("u", "", "username")
	passPtr := flag.String("p", "", "password")
	stationPtr := flag.Uint("s", 0, "station nubmer")
	listenPtr := flag.String("http", "0.0.0.0:7890", "listen address")
	flag.Parse()

	if len(*usernamePtr) == 0 {
		fmt.Println("Must provide username (-u)")
		os.Exit(1)
	}

	if len(*passPtr) == 0 {
		fmt.Println("Must provide password (-p)")
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

	http.HandleFunc("/", getWebHandler(commands))
	log.Printf("Listening on %s, playing station: %d", *listenPtr, *stationPtr)
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

func getWebHandler(commands chan<- string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		command := r.URL.Query().Get("cmd")
		if len(command) > 0 {
			log.Printf("Got command: %q", command)
			switch command {
			case "pause":
				commands <- " "
			case "next":
				commands <- "n"
			case "fav":
				commands <- "+"
			case "ban":
				commands <- "-"
			case "volup1":
				commands <- ")"
			case "volup3":
				commands <- ")))"
			case "voldown1":
				commands <- "("
			case "voldown3":
				commands <- "((("
			default:
				log.Printf("Unrecognized command: %q", command)
			}
		}

		fmt.Fprintf(w, `
		<html>
		<head>
			<title>pianobar-web</title>
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				body {
					margin: 0;
					padding: 0;
				}

				a {
					font-size: 1em;
					text-decoration: none;
					color: black;
					display: inline-block;
					padding: 12%% 0 0 0;
					margin: 0 0 0.3em 0;
					font-weight: bold;
					width: 48%%;
					height: 15.5%%;
					text-align: center;
					vertical-align: middle;
				}
			</style>
		</head>
		<body>
			<a href="/?cmd=pause" style="background-color: yellow;">(un)pause</a>
			<a href="/?cmd=next" style="background-color: orange;">next</a>
			<a href="/?cmd=fav" style="background-color: pink;">fav</a>
			<a href="/?cmd=ban" style="background-color: red;">ban</a>
			<a href="/?cmd=volup1" style="background-color: green;">vol up</a>
			<a href="/?cmd=volup3" style="background-color: lightgreen;">up x3</a>
			<a href="/?cmd=voldown1" style="background-color: lightblue;">vol down</a>
			<a href="/?cmd=voldown3" style="background-color: blue;">down x3</a>
		</body>
		</html>`)
		// TODO: pre-selected commands, display UI
	}
}
