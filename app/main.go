package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Value struct {
	data      string
	expiresAt time.Time
}

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	store := make(map[string]Value)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleThisClient(conn, store)
	}
}

func handleThisClient(conn net.Conn, store map[string]Value) {
	defer conn.Close()
	buf := make([]byte, 1024)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}

		data := string(buf[:n])
		parts, err := parseRESP(data)
		if err != nil {
			fmt.Println("Error parsing RESP:", err)
			continue
		}

		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])

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
			val := parts[2]

			var expiryTime time.Time

			if len(parts) >= 5 {
				expiryType := strings.ToUpper(parts[3])

				if expiryType == "PX" {
					milliseconds, err := strconv.Atoi(parts[4])
					if err == nil {
						expiryTime = time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
					}
				} else if expiryType == "EX" {
					seconds, err := strconv.Atoi(parts[4])
					if err == nil {
						expiryTime = time.Now().Add(time.Duration(seconds) * time.Second)
					}
				}
			}

			store[key] = Value{
				data:      val,
				expiresAt: expiryTime,
			}
			conn.Write([]byte("+OK\r\n"))
		} else if command == "GET" {
			key := parts[1]
			val, exists := store[key]

			if !exists {
				conn.Write([]byte("$-1\r\n"))
				continue
			}

			if !val.expiresAt.IsZero() && time.Now().After(val.expiresAt) {
				delete(store, key)
				conn.Write([]byte("$-1\r\n"))
				continue
			}

			length := len(val.data)
			response := fmt.Sprintf("$%d\r\n%s\r\n", length, val.data)
			conn.Write([]byte(response))
		}
	}
}

func parseRESP(data string) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	if data[0] != '*' {
		return nil, fmt.Errorf("expected array, got: %c", data[0])
	}

	parts := strings.Split(data, "\r\n")
	arrayLengthStr := parts[0][1:]
	arrayLength, err := strconv.Atoi(arrayLengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %s", arrayLengthStr)
	}

	result := make([]string, 0, arrayLength)
	idx := 1

	for i := 0; i < arrayLength; i++ {
		if idx >= len(parts) {
			return nil, fmt.Errorf("unexpected end of data")
		}

		idx++

		if idx >= len(parts) {
			return nil, fmt.Errorf("unexpected end of data")
		}
		result = append(result, parts[idx])
		idx++
	}

	return result, nil
}
