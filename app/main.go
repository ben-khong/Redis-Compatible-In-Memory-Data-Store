package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment the code below to pass the first stage

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleThisClient(conn)
	}
}

func handleThisClient(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}

		data := string(buf[:n])

		// Parse the RESP data
		parts, err := parseRESP(data)
		if err != nil {
			fmt.Println("Error parsing RESP:", err)
			continue
		}

		// Now parts[0] is the command, parts[1], parts[2]... are arguments
		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])
		// Create map to set a key to a value
		m := make(map[string]string)

		if command == "PING" {
			conn.Write([]byte("+PONG\r\n"))
		} else if command == "ECHO" {
			if len(parts) < 2 {
				conn.Write([]byte("-ERR wrong number of arguments\r\n"))
				continue
			}
			argument := parts[1]
			length := len(argument)
			response := fmt.Sprintf("$%d\r\n%s\r\n", length, argument)
			conn.Write([]byte(response))
		} else if command == "SET" {
			key := parts[1]
			value := parts[2]
			m[key] = value
			conn.Write([]byte("+OK\r\n"))
		} else if command == "GET" {
			key := parts[1]
			value := m[key]
			length := len(value)
			response := fmt.Sprintf("$%d\r\n%s\r\n", length, value)
			conn.Write([]byte(response))
		}

	}
}

func parseRESP(data string) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	// Check if it's an array
	if data[0] != '*' {
		return nil, fmt.Errorf("expected array, got: %c", data[0])
	}

	// Split by \r\n to get parts
	parts := strings.Split(data, "\r\n")

	// Parse array length from first element (e.g., "*2" -> 2)
	arrayLengthStr := parts[0][1:] // Skip the '*'
	arrayLength, err := strconv.Atoi(arrayLengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %s", &arrayLengthStr)
	}

	// Extract the actual strings from the array
	result := make([]string, 0, arrayLength)
	idx := 1 // Start after "*2"

	for i := 0; i < arrayLength; i++ {
		// Each element has format: $<length>\r\n<string>
		if idx >= len(parts) {
			return nil, fmt.Errorf("unexpected end of data")
		}

		// Skip the length indicator (e.g., "$4")
		idx++

		// Get the actual string
		if idx >= len(parts) {
			return nil, fmt.Errorf("unexpected end of data")
		}
		result = append(result, parts[idx])
		idx++
	}

	return result, nil

}
