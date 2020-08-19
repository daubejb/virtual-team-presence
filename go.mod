module github.com/daubejb/virtual-team-presence

require (
	github.com/gorilla/mux latest
  github.com/rs/cors latest
	golang.org/x/oauth2 latest
	golang.org/x/oauth2/google latest
	golang.org/x/oauth2/jwt latest
	google.golang.org/api/calendar/v3 latest
)

go 1.14

require golang.org/x/tools v0.0.0-20200819193742-d088b475e336 // indirect
