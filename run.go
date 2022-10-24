package main

import (
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
)

func main() {

	// Handle SIGINT
	// Clear execution environment
	go func() {
		sigchan := make(chan os.Signal)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan
		fmt.Println("Program killed!")

		// Exec command 'docker compose down'
		cmd := exec.Command("cmd.exe", "/c", "start", "docker", "compose", "down")
		err := cmd.Start()
		if err != nil {
			log.Fatalln("Command exec error: ", err)
		}

		// Exec command 'docker rmi all'
		out, _ := exec.Command("cmd.exe", "/c", "docker", "images", "-a", "-q").Output()
		for _, img := range out {
			cmd = exec.Command("cmd.exe", "/c", "docker", "rmi", string(img))
			err = cmd.Start()
			if err != nil {
				log.Fatalln("Command exec error: ", err)
			}
		}

		// Delete .env file
		err = os.Remove(".env")
		if err != nil {
			log.Fatalln("Remove error: ", err)
		}

		os.Exit(0)
	}()

	// Set application flags
	aflag := flag.String("a", "", "Election algorithm")
	nflag := flag.Int("n", 0, "Number of peers")
	dflag := flag.Int("d", 2, "Delay in seconds to send a message")
	rflag := flag.Int("r", 3, "Number of tries to send a message")
	hbflag := flag.Int("hb", 2, "Heartbeat repeat time")
	vflag := flag.Bool("v", false, "Verbose")

	// Retrieve flags value
	flag.Parse()

	// Check correctness of flags
	if *nflag <= 0 || (*aflag != "bully" && *aflag != "ring") {
		flag.Usage()
		os.Exit(0)
	}

	// Create and open .env file
	file, err := os.Create(".env")
	if err != nil {
		log.Fatalln("Crate error: ", err)
	}

	// Load .env file
	err = godotenv.Load()
	if err != nil {
		log.Fatalln("Load env error: ", err)
	}

	mp := make(map[string]string)

	// Set number of peers in .env file
	mp["PEERS"] = strconv.Itoa(*nflag)

	// Set VERBOSE in .env file
	if *vflag {
		mp["VERBOSE"] = "1"
	}

	// Set hbtime in .env file
	mp["HEARTBEAT"] = strconv.Itoa(*hbflag)

	// Set algorithm type in .env file
	mp["ALGO"] = *aflag

	// Set delay in .env file
	mp["DELAY"] = strconv.Itoa(*dflag)

	// Set tries in .env file
	mp["TRIES"] = strconv.Itoa(*rflag)

	// Write .env file
	err = godotenv.Write(mp, ".env")
	if err != nil {
		log.Fatalln("Write env error: ", err)
	}

	// Close .env file
	err = file.Close()
	if err != nil {
		log.Fatalln("Close env error: ", err)
	}

	// Exec command 'docker compose build'
	cmd := exec.Command("cmd.exe", "/c", "docker", "compose", "build")
	err = cmd.Start()
	if err != nil {
		log.Fatalln("Command exec error: ", err)
	}

	// Exec command 'docker compose up'
	cmd = exec.Command("cmd.exe", "/c", "start", "docker", "compose", "up")
	err = cmd.Start()
	if err != nil {
		log.Fatalln("Command exec error: ", err)
	}

	select {}
}
