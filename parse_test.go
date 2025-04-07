package main

import "testing"

func Test_parseServer(t *testing.T) {
	testcases := []struct {
		name    string
		server  string
		wantOut Server
	}{
		{
			name:   "FQDN Port",
			server: "candinya.com:2233",
			wantOut: Server{
				Host: "candinya.com",
				Port: 2233,
			},
		},
		{
			name:   "FQDN only",
			server: "candinya.com",
			wantOut: Server{
				Host: "candinya.com",
				Port: 22,
			},
		},
		{
			name:   "IPv4 Port",
			server: "127.0.0.1:2233",
			wantOut: Server{
				Host: "127.0.0.1",
				Port: 2233,
			},
		},
		{
			name:   "IPv4 only",
			server: "127.0.0.1",
			wantOut: Server{
				Host: "127.0.0.1",
				Port: 22,
			},
		},
		{
			name:   "IPv6 Port",
			server: "[fe80::1]:2233",
			wantOut: Server{
				Host: "fe80::1",
				Port: 2233,
			},
		},
		{
			name:   "IPv6 only",
			server: "[fe80::1]",
			wantOut: Server{
				Host: "fe80::1",
				Port: 22,
			},
		},
		{
			name:   "IPv6 raw",
			server: "fe80::1",
			wantOut: Server{
				Host: "fe80::1",
				Port: 22,
			},
		},
		{
			name:   "FQDN Port Username",
			server: "candinya@candinya.com:2233",
			wantOut: Server{
				Username: p("candinya"),
				Host:     "candinya.com",
				Port:     2233,
			},
		},
		{
			name:   "FQDN Username",
			server: "candinya@candinya.com",
			wantOut: Server{
				Username: p("candinya"),
				Host:     "candinya.com",
				Port:     22,
			},
		},
		{
			name:   "IPv4 Port Username Password",
			server: "candinya:password@127.0.0.1:2233",
			wantOut: Server{
				Username: p("candinya"),
				Password: p("password"),
				Host:     "127.0.0.1",
				Port:     2233,
			},
		},
		{
			name:   "IPv6 Username Password",
			server: "candinya:password@[fe80::1]",
			wantOut: Server{
				Username: p("candinya"),
				Password: p("password"),
				Host:     "fe80::1",
				Port:     22,
			},
		},
		{
			name:   "IPv6 Port Username Password",
			server: "candinya:password@[fe80::1]:2233",
			wantOut: Server{
				Username: p("candinya"),
				Password: p("password"),
				Host:     "fe80::1",
				Port:     2233,
			},
		},
		{
			name:   "IPv6 raw Username Password",
			server: "candinya:password@fe80::1",
			wantOut: Server{
				Username: p("candinya"),
				Password: p("password"),
				Host:     "fe80::1",
				Port:     22,
			},
		},
		{
			name:   "FQDN Username Password colon",
			server: "candinya:pass:word@candinya.com",
			wantOut: Server{
				Username: p("candinya"),
				Password: p("pass:word"),
				Host:     "candinya.com",
				Port:     22,
			},
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// Exec and collect error
			gotOut, err := parseServer(testcase.server)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Compare out
			if testcase.wantOut.Username != nil {
				if gotOut.Username == nil {
					t.Errorf("Unexpected nil username")
				} else if *testcase.wantOut.Username != *gotOut.Username {
					t.Errorf("Unexpected username: expected %q, got %q", *testcase.wantOut.Username, *gotOut.Username)
				}
			} else if gotOut.Username != nil {
				t.Errorf("Unexpected username: got %q", *gotOut.Username)
			}

			if testcase.wantOut.Password != nil {
				if gotOut.Password == nil {
					t.Errorf("Unexpected nil password")
				} else if *testcase.wantOut.Password != *gotOut.Password {
					t.Errorf("Unexpected password: expected %q, got %q", *testcase.wantOut.Password, *gotOut.Password)
				}
			} else if gotOut.Password != nil {
				t.Errorf("Unexpected password: got %q", *gotOut.Password)
			}

			if testcase.wantOut.Host != gotOut.Host {
				t.Errorf("Unexpected host: expected %q, got %q", testcase.wantOut.Host, gotOut.Host)
			}

			if testcase.wantOut.Port != gotOut.Port {
				t.Errorf("Unexpected port: expected %d, got %d", testcase.wantOut.Port, gotOut.Port)
			}
		})
	}
}
