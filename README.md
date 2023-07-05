# GoArista Script

This script is written in Go and allows you to connect to Arista switches, execute show commands, and save the output to files. The script takes input from a CSV file containing a list of IP addresses and hostnames for the switches.

## Prerequisites

- Go programming language (installed and configured)
- Arista switches with SSH enabled

## Usage

1. Clone the repository and navigate to the project directory.

2. Prepare the CSV file:
   - Create a CSV file with the IP addresses and hostnames of the Arista switches.
   - The CSV file should have two columns:
     - Column A: Hostnames
     - Column B: IP addresses
   - Save the CSV file as `ip_addresses.csv` (or provide a custom file name/path when prompted).

3. Prepare the show commands CSV file:
   - Create a CSV file with the show commands you want to execute on the switches.
   - The CSV file should have a single row containing the show commands.
   - Save the CSV file as `show_commands.csv` (or provide a custom file name/path when prompted).

4. Run the script:
   - Open a terminal and navigate to the project directory.
   - Run the command: `go run main.go`
   - Follow the prompts:
     - Enter the SSH username and password.
     - Enter the path to the CSV file (or press Enter for default: `ip_addresses.csv`).
     - Enter the path to the CSV file for show commands (or press Enter for default: `show_commands.csv`).

5. The script will connect to each switch, execute the show commands, and save the output to individual files.
   - The output files will be named `<hostname>_<timestamp>_output.txt`.
   - The timestamp is appended to ensure each file has a unique name.

## Customization

- To modify the default CSV file names (`ip_addresses.csv` and `show_commands.csv`), you can provide custom file names or paths when prompted.
- You can update the default show commands by editing the `show_commands.csv` file.

## Limitations

- The script currently supports authentication using a username and password. If you need to use other authentication methods, additional modifications may be required.

## License

This project is licensed under the [MIT License](LICENSE).

