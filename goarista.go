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
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

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

	// Read the IP addresses and hostnames from the CSV file
	ipAddresses, hostnames, err := readIPAddressesAndHostnames(file)
	if err != nil {
		log.Fatal(err)
	}

	// Get the SSH username and password from the user
	username := prompt("Enter the SSH username: ")
	password := promptPassword("Enter the SSH password: ")

	// Get the show commands from the CSV file or use the default commands
	showCommandsFilePath := prompt("Enter the path to the CSV file for show commands (or press Enter for default: show_commands.csv): ")
	if showCommandsFilePath == "" {
		showCommandsFilePath = "show_commands.csv"
	}

	// Look for the show commands CSV file in the current directory if not provided with a full path
	if !filepath.IsAbs(showCommandsFilePath) {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		showCommandsFilePath = filepath.Join(currentDir, showCommandsFilePath)
	}

	// Read the show commands from the CSV file or use the default commands
	showCommands, err := readShowCommands(showCommandsFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Validate show commands
	if !validateShowCommands(showCommands) {
		log.Fatal("One or more show commands do not contain the word 'show'. Please check the commands.")
	}

	// Create a timestamped folder in the current directory
	timestamp := time.Now().Format("06_01_02_150405")
	outputDir := filepath.Join(".", timestamp)
	err = os.Mkdir(outputDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// WaitGroup to wait for all Goroutines to finish
	var wg sync.WaitGroup

	// Concurrently connect to each IP address and execute show commands
	for i, ipAddress := range ipAddresses {
		wg.Add(1)
		go func(ip, hostname string) {
			defer wg.Done()

			// Connect to the Arista switch via SSH
			connection, err := connectSSH(ip, username, password)
			if err != nil {
				log.Printf("Failed to connect to %s (%s): %v", hostname, ip, err)
				return
				// We use return instead of continue to skip the rest of the Goroutine
			}
			defer connection.Close()

			// Execute show commands
			output, err := executeShowCommandsSSH(connection, showCommands)
			if err != nil {
				log.Printf("Failed to execute show commands on %s (%s): %v", hostname, ip, err)
				return
				// We use return instead of continue to skip the rest of the Goroutine
			}

			// Generate output filename using the hostname and timestamp
			outputFilePath := filepath.Join(outputDir, fmt.Sprintf("%s_output.txt", hostname))

			// Write output to a file
			err = writeToFile(outputFilePath, output)
			if err != nil {
				log.Printf("Failed to write output to file for %s (%s): %v", hostname, ip, err)
				return
				// We use return instead of continue to skip the rest of the Goroutine
			}

			fmt.Printf("Output written to %s\n", outputFilePath)
		}(ipAddress, hostnames[i])
	}

	// Wait for all Goroutines to finish
	wg.Wait()
}

func readShowCommands(csvFilePath string) ([]string, error) {
	// Open the CSV file
	file, err := os.Open(csvFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the show commands from the CSV file
	reader := csv.NewReader(file)
	showCommands, err := reader.Read()
	if err != nil {
		return nil, err
	}

	// Trim leading and trailing spaces from each command
	for i := range showCommands {
		showCommands[i] = strings.TrimSpace(showCommands[i])
	}

	return showCommands, nil
}

// ValidateShowCommands checks if all show commands contain the word "show"
func validateShowCommands(showCommands []string) bool {
	for _, command := range showCommands {
		if !strings.Contains(command, "show") {
			return false
		}
	}
	return true
}

func readIPAddressesAndHostnames(reader io.Reader) ([]string, []string, error) {
	ipAddresses := make([]string, 0)
	hostnames := make([]string, 0)

	csvReader := csv.NewReader(reader)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		if len(record) >= 2 {
			ipAddresses = append(ipAddresses, strings.TrimSpace(record[1]))
			hostnames = append(hostnames, strings.TrimSpace(record[0]))
		}
	}

	return ipAddresses, hostnames, nil
}

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
