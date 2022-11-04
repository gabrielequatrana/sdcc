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
	"strings"
)

func main() {

	// Handle SIGINT
	// Clear execution environment
	go func() {
		sigCh := make(chan os.Signal)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		fmt.Println("Program killed!")

		// Exec command 'docker compose down'
		cmd := exec.Command("cmd.exe", "/c", "start", "docker", "compose", "down")
		_, err := cmd.Output()
		if err != nil {
			log.Fatalln("Command exec error: ", err)
		}

		// Exec command 'docker rmi all'
		out, err := exec.Command("cmd.exe", "/c", "docker", "images", "-a", "-q").Output()
		if err != nil {
			log.Fatalln("Command exec error: ", err)
		}

		// Delete all images
		for i := 0; i < len(out); i += 13 {
			cmd = exec.Command("cmd.exe", "/c", "docker", "rmi", string(out[i:i+12]))
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
	aFlag := flag.String("a", "", "Election algorithm (select \"bully\" or \"ring\")")
	nFlag := flag.Int("n", 0, "Number of peers (at least 2)")
	dFlag := flag.Int("d", 2, "Delay in seconds to send a message")
	rFlag := flag.Int("r", 3, "Number of tries to send a message")
	hbFlag := flag.Int("hb", 2, "Heartbeat repeat time in seconds")
	vFlag := flag.Bool("v", false, "Print debug information")

	// Retrieve flags value
	flag.Parse()

	// Check correctness of flags
	*aFlag = strings.ToLower(*aFlag)
	if *nFlag <= 1 || (*aFlag != "bully" && *aFlag != "ring") {
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

	// map used to add flags to .env file
	mp := make(map[string]string)

	// Set number of peers in .env file
	mp["PEERS"] = strconv.Itoa(*nFlag)

	// Set VERBOSE in .env file
	if *vFlag {
		mp["VERBOSE"] = "1"
	}

	// Set hbTime in .env file
	mp["HEARTBEAT"] = strconv.Itoa(*hbFlag)

	// Set algorithm type in .env file
	mp["ALGO"] = *aFlag

	// Set delay in .env file
	mp["DELAY"] = strconv.Itoa(*dFlag)

	// Set tries in .env file
	mp["TRIES"] = strconv.Itoa(*rFlag)

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
