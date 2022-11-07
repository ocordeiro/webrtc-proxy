package main

import "os"

type Message struct {
	Type string `json:"type"`
}

func main() {

	arg := os.Args[1]

	if arg == "server" {
		Server()
	}

	if arg == "client" {
		Client()
	}
}
