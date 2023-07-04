package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

var showCommands = []string{
	"show version",
	"show interfaces",
	"show ip route",
}

func main() {
	printBanner()

	// Get the CSV file path from the user or use the default file name "ip_addresses.csv"
	csvFilePath := prompt("Enter the path to the CSV file (or press Enter for default: ip_addresses.csv): ")
	if csvFilePath == "" {
		csvFilePath = "ip_addresses.csv"
	}

	// Look for the CSV file in the current directory if not provided with a full path
	if !filepath.IsAbs(csvFilePath) {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		csvFilePath = filepath.Join(currentDir, csvFilePath)
	}

	// Open the CSV file
	file, err := os.Open(csvFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read the IP addresses from the CSV file
	ipAddresses, err := readIPAddresses(file)
	if err != nil {
		log.Fatal(err)
	}

	// Get the SSH username and password from the user
	username := prompt("Enter the SSH username: ")
	password := promptPassword("Enter the SSH password: ")

	// WaitGroup to wait for all Goroutines to finish
	var wg sync.WaitGroup

	// Concurrently connect to each IP address and execute show commands
	for _, ipAddress := range ipAddresses {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			// Connect to the Arista switch via SSH
			connection, err := connectSSH(ip, username, password)
			if err != nil {
				log.Printf("Failed to connect to %s: %v", ip, err)
				return
				// We use return instead of continue to skip the rest of the Goroutine
			}
			defer connection.Close()

			// Execute show commands
			output, err := executeShowCommandsSSH(connection, showCommands)
			if err != nil {
				log.Printf("Failed to execute show commands on %s: %v", ip, err)
				return
				// We use return instead of continue to skip the rest of the Goroutine
			}

			// Write output to a file
			outputFilePath := fmt.Sprintf("%s_output.txt", ip)
			err = writeToFile(outputFilePath, output)
			if err != nil {
				log.Printf("Failed to write output to file for %s: %v", ip, err)
				return
				// We use return instead of continue to skip the rest of the Goroutine
			}

			fmt.Printf("Output written to %s\n", outputFilePath)
		}(ipAddress)
	}

	// Wait for all Goroutines to finish
	wg.Wait()
}

// Rest of the code...

func printBanner() {
	banner := `
 -----                     _     _        
/ ____|         /\        (_)   | |       
| |  __  ___   /  \   _ __ _ ___| |_ __ _ 
| | |_ |/ _ \ / /\ \ | '__| / __| __/ _  |
| |__| | (_) / ____ \| |  | \__ \ || (_| |
 \_____|\___/_/    \_\_|  |_|___/\__\__,_|
											 
											    
`

	fmt.Println(banner)
}

func readIPAddresses(file io.Reader) ([]string, error) {
	var ipAddresses []string

	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		ipAddress := strings.TrimSpace(record[0])
		ipAddresses = append(ipAddresses, ipAddress)
	}

	return ipAddresses, nil
}

func connectSSH(ipAddress, username, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				answers = make([]string, len(questions))
				for i := range questions {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := ssh.Dial("tcp", ipAddress+":22", config)
	if err != nil {
		return nil, err
	}

	return connection, nil
}

func executeShowCommandsSSH(connection *ssh.Client, commands []string) (string, error) {
	var output strings.Builder

	for _, command := range commands {
		session, err := connection.NewSession()
		if err != nil {
			return "", err
		}

		out, err := session.Output(command)
		if err != nil {
			session.Close()
			return "", err
		}

		output.WriteString(fmt.Sprintf("Command: %s\n", command))
		output.Write(out)
		output.WriteString("\n")

		session.Close()
	}

	return output.String(), nil
}

func writeToFile(filePath, data string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}

func prompt(message string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print(message)
	input, _ := reader.ReadString('\n')

	return strings.TrimSpace(input)
}

func promptPassword(message string) string {
	fmt.Print(message)

	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()

	return string(bytePassword)
}
