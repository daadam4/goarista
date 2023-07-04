package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func main() {
	// Get the CSV file path from the user
	csvFilePath := prompt("Enter the path to the CSV file: ")

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

	// Get the show commands from the user or use default commands
	var showCommands string
	useDefaultCommands := promptYesNo("Do you want to use default show commands? (Y/N): ")
	if useDefaultCommands {
		showCommands = getDefaultShowCommands()
	} else {
		showCommands = prompt("Enter the show commands (comma-separated): ")
	}

	// Connect to each IP address and execute show commands
	for _, ipAddress := range ipAddresses {
		// Connect to the Arista switch via SSH
		connection, err := connectSSH(ipAddress, username, password)
		if err != nil {
			log.Printf("Failed to connect to %s: %v", ipAddress, err)
			continue
		}
		defer connection.Close()

		// Execute show commands
		output, err := executeShowCommandsSSH(connection, showCommands)
		if err != nil {
			log.Printf("Failed to execute show commands on %s: %v", ipAddress, err)
			continue
		}

		// Write output to a file
		outputFilePath := fmt.Sprintf("%s_output.txt", ipAddress)
		err = writeToFile(outputFilePath, output)
		if err != nil {
			log.Printf("Failed to write output to file for %s: %v", ipAddress, err)
			continue
		}

		fmt.Printf("Output written to %s\n", outputFilePath)
	}
}

func readIPAddresses(file io.Reader) ([]string, error) {
	var ipAddresses []string

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read all IP addresses from the CSV file
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Assume the IP address is in the first column
		ipAddress := strings.TrimSpace(record[0])
		ipAddresses = append(ipAddresses, ipAddress)
	}

	return ipAddresses, nil
}

func connectSSH(ipAddress, username, password string) (*ssh.Client, error) {
	// Define the SSH configuration
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the Arista switch
	connection, err := ssh.Dial("tcp", ipAddress+":22", config)
	if err != nil {
		return nil, err
	}

	return connection, nil
}

func executeShowCommandsSSH(connection *ssh.Client, showCommands string) (string, error) {
	// Create a session
	session, err := connection.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	// Split show commands by comma
	commands := strings.Split(showCommands, ",")

	// Execute show commands
	var output strings.Builder
	for _, command := range commands {
		command = strings.TrimSpace(command)

		// Run the command
		outputBytes, err := session.CombinedOutput(command)
		if err != nil {
			return "", err
		}

		// Append the output to the overall result
		output.WriteString(fmt.Sprintf("\nOutput for '%s':\n\n%s\n", command, string(outputBytes)))
	}

	return output.String(), nil
}

func writeToFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(content)
	if err != nil {
		return err
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func prompt(question string) string {
	fmt.Print(question + " ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func promptPassword(question string) string {
	fmt.Print(question + " ")
	password, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return string(password)
}

func promptYesNo(question string) bool {
	for {
		response := strings.ToLower(prompt(question))
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		} else {
			fmt.Println("Invalid response. Please enter 'Y' or 'N'.")
		}
	}
}

func getDefaultShowCommands() string {
	// Set your default show commands here
	defaultCommands := []string{
		"show version",
		"show interfaces",
		"show ip route",
	}

	return strings.Join(defaultCommands, ",")
}
