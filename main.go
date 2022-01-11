package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// TODO: catch signal and tell pianobar to quit (enter, enter, q, enter)
// TODO: scrape channel list on start, will use to populate dropdown.
// TODO: refactor so channel based comms
// TODO: put web in front for skip, volume up, volume down, +/-, change station
func main() {

	usernamePtr := flag.String("u", "", "username")
	passPtr := flag.String("p", "", "password")
	stationPtr := flag.Uint("s", 0, "station nubmer")
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

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		log.Fatal(err)
	}

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

	x := 0
	for {
		<-time.After(40 * time.Second)
		stdin.Write([]byte("n\n"))
		x += 1
		if x > 10 {
			break
		}
	}

	data, err := ioutil.ReadAll(stdout)

	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Output: %q\n", string(data))
}
