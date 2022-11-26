package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var crash []int  // Peer that will crash in test mode
var shell string // Shell used to run the program
var arg string   // Shell argument
var cFlag *bool  // If true clean the environment after the execution

func main() {

	// Get stdout to flush after
	std := bufio.NewWriter(os.Stdout)

	// Check if the OS is Windows or Linux
	OS := runtime.GOOS
	switch OS {
	case "windows":
		fmt.Println("Running on Windows")
		shell = "cmd.exe"
		arg = "/c"

	case "linux":
		fmt.Println("Running on Linux")
		shell = "/bin/sh"
		arg = "-c"
	}

	// Handle SIGINT
	go func() {
		sigCh := make(chan os.Signal)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		fmt.Println("Program killed!")

		// Flush stdout
		err := std.Flush()
		if err != nil {
			log.Fatalln("Flush error 3:", err)
		}

		// Exec command 'docker compose down'
		cmd := exec.Command(shell, arg, "docker compose down")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err = cmd.Run()
		if err != nil {
			log.Fatalln("Command exec error 3:", err)
		}

		if *cFlag {
			// Exec command 'docker images'
			out, err2 := exec.Command(shell, arg, "docker", "images", "-a", "-q").Output()
			if err2 != nil {
				log.Fatalln("Command exec error: ", err2)
			}

			// Delete the images
			for i := 0; i < 26; i += 13 {
				cmd = exec.Command(shell, arg, "docker", "rmi", string(out[i:i+12]))
				err2 = cmd.Start()
				if err2 != nil {
					log.Fatalln("Command exec error: ", err2)
				}
			}
		}

		// Delete .env file
		err = os.Remove(".env")
		if err != nil {
			log.Fatalln("Remove error:", err)
		}

		os.Exit(0)
	}()

	// Set application flags
	aFlag := flag.String("a", "", "Election algorithm (select \"bully\" or \"ring\")")
	nFlag := flag.Int("n", 0, "Number of peers (at least 2)")
	dFlag := flag.Int("d", 1000, "Delay in ms to send a message")
	hbFlag := flag.Int("hb", 2, "Heartbeat repeat time in seconds")
	vFlag := flag.Bool("v", false, "Print some debug information")
	vvFlag := flag.Bool("vv", false, "Print all debug information")
	cFlag = flag.Bool("c", false, "Clean the images after the execution")
	tFlag := flag.Int("t", 0, "Execute a test")

	// Retrieve flags value
	flag.Parse()

	// map used to add flags to .env file
	mp := make(map[string]string)

	// Check correctness of flags
	*aFlag = strings.ToLower(*aFlag)
	if *nFlag <= 1 || (*aFlag != "bully" && *aFlag != "ring") || *tFlag >= 4 {
		flag.Usage()
		os.Exit(0)
	}

	// Check if executing test
	if *tFlag != 0 {

		// At least 4 peers to run tests
		if *nFlag <= 3 {
			flag.Usage()
			os.Exit(0)
		}

		// Set randomizer seed
		rand.Seed(time.Now().UnixNano())

		// Check test type
		switch *tFlag {

		// Crash one non coordinator peer
		case 1:
			crash = append(crash, rand.Intn(*nFlag-1))
			fmt.Println("Running Test 1 with", *nFlag, "peers. The peer", crash[0], "will crash")
			mp["CRASH"] = strconv.Itoa(crash[0])

		// Crash the coordinator peer
		case 2:
			crash = append(crash, *nFlag-1)
			fmt.Println("Running Test 2 with", *nFlag, "peers. The coordinator peer will crash")
			mp["CRASH"] = strconv.Itoa(crash[0])

		// Crash at least one non coordinator peer and the coordinator peer
		case 3:
			num := rand.Intn(*nFlag - 1)
			for i := 0; i <= num; i++ {
				p := rand.Intn(*nFlag - 1)
				if !search(crash, p) {
					crash = append(crash, p)
				} else {
					i--
				}
			}
			crash = append(crash, *nFlag-1)
			sort.Ints(crash)
			fmt.Println("Running Test 3 with", *nFlag, "peers. The peers", crash[:len(crash)-1], ""+
				"and the coordinator will crash")

			mp["CRASH"] = strconv.Itoa(crash[0])
			for i := 1; i < len(crash); i++ {
				mp["CRASH"] = mp["CRASH"] + ";" + strconv.Itoa(crash[i])
			}

		}

	} else {
		// Non test mode
		mp["CRASH"] = "-1"
	}

	// Create and open .env file
	file, err := os.Create(".env")
	if err != nil {
		log.Fatalln("Crate error:", err)
	}

	// Load .env file
	err = godotenv.Load()
	if err != nil {
		log.Fatalln("Load env error:", err)
	}

	// Set number of peers in .env file
	mp["PEERS"] = strconv.Itoa(*nFlag)

	// Set verbose in .env file
	if *vFlag {
		mp["VERBOSE"] = "1"
	}

	// Set verbose in .env file
	if *vvFlag {
		mp["VERBOSE"] = "2"
	}

	// Set hbTime in .env file
	mp["HEARTBEAT"] = strconv.Itoa(*hbFlag)

	// Set algorithm type in .env file
	mp["ALGO"] = *aFlag

	// Set delay in .env file
	mp["DELAY"] = strconv.Itoa(*dFlag)

	// Write .env file
	err = godotenv.Write(mp, ".env")
	if err != nil {
		log.Fatalln("Write env error:", err)
	}

	// Close .env file
	err = file.Close()
	if err != nil {
		log.Fatalln("Close env error:", err)
	}

	// Exec command 'docker compose build'
	cmd := exec.Command(shell, arg, "docker compose build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		log.Fatalln("Command exec error 1:", err)
	}

	// Flush stdout
	err = std.Flush()
	if err != nil {
		log.Fatalln("Flush error 1:", err)
	}

	// Exec command 'docker compose up'
	cmd = exec.Command(shell, arg, "docker compose up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil && err.Error() != "exit status 130" {
		log.Fatalln("Command exec error 2:", err)
	}

	// Flush stdout
	err = std.Flush()
	if err != nil {
		log.Fatalln("Flush error 2:", err)
	}

	select {}
}

// Search an int from a slice of int
func search(slice []int, j int) bool {
	for i := 0; i <= len(slice)-1; i++ {
		if slice[i] == j {
			return true
		}
	}
	return false
}
