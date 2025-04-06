package main

type Server struct {
	// Authentication
	Username *string
	Password *string

	// SSH server
	Host string
	Port int
}
