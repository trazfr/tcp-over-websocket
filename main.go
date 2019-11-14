package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func globalUsage() {
	fmt.Fprintln(os.Stderr, os.Args[0], "<type> [arguments]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "type:")
	fmt.Fprintln(os.Stderr, " - server: HTTP websocket server which connectes to a TCP server")
	fmt.Fprintln(os.Stderr, " - client: TCP server which connects to a websocket server")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Example: connect to ssh-server through an HTTP proxy running on ws-server")
	fmt.Fprintln(os.Stderr, "-", os.Args[0], "server -listen_ws :8080 -connect_tcp ssh-server.example.org:22")
	fmt.Fprintln(os.Stderr, "-", os.Args[0], "client -listen_tcp 127.0.0.1:1234 -connect_ws ws://ws-server.example.org:8080/")
	os.Exit(1)
}

func createHTTPServer(args []string) Runner {
	listen, connect := "", ""

	fs := flag.NewFlagSet("server", flag.ExitOnError)
	fs.StringVar(&listen, "listen_ws", "",
		"Local address to listen to\n"+
			"Examples: \":8080\", \"127.0.0.1:1234\", \"[::1]:5000\"")
	fs.StringVar(&connect, "connect_tcp", "",
		"Remote address to connect to at each incoming websocket connection\n"+
			"Examples: \"127.0.0.1:23\", \"ssh.example.com:22\", \"[::1]:143\"")
	fs.Parse(args)

	if listen == "" || connect == "" {
		fs.Usage()
		os.Exit(1)
	}

	return NewHTTPServer(listen, connect)
}

func createHTTPClient(args []string) Runner {
	listen, connect := "", ""

	fs := flag.NewFlagSet("client", flag.ExitOnError)
	fs.StringVar(&listen, "listen_tcp", "",
		"Local address to listen to\n"+
			"Examples: \":8080\", \"127.0.0.1:1234\", \"[::1]:5000\")")
	fs.StringVar(&connect, "connect_ws", "",
		"Remote websocket to connect to at each incoming TCP connection \n"+
			"Examples: \"ws://192.168.0.1:8080/\", \"wss://https.example.org/\", \"ws://[::1]/\"\n"+
			"If the server is behind a reverse proxy, it may be something like: \"wss://https.example.org/fragment/\"")
	fs.Parse(args)

	if listen == "" || connect == "" {
		fs.Usage()
		os.Exit(1)
	}

	return NewHTTPClient(listen, connect)
}

func create() Runner {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "server":
			return createHTTPServer(os.Args[2:])
		case "client":
			return createHTTPClient(os.Args[2:])
		}
	}
	globalUsage()
	return nil
}

func main() {
	err := create().Run()
	if err != nil {
		log.Fatalf("Failed to run: %s", err)
	}
}
