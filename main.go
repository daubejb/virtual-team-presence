package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var (
	//	Flags
	port           = flag.Int("port", 8080, "The port to listen on")
	authEmail      = flag.String("authEmail", "calendar@virtual-team-presence-81.iam.gserviceaccount.com", "Service account email address")
	authSubject    = flag.String("authSubject", "jedaube@redhat.com", "Impersonated user email address")
	allowedOrigins = flag.String("allowedOrigins", "*", "A comma-separated list of valid CORS origins")
)

func parseEnvironment() {
	//	Check for the listen port
	if envPort := os.Getenv("CALENDAR_PORT"); envPort != "" {
		*port, _ = strconv.Atoi(envPort)
	}

	//	Check for allowed origins
	if envOrigins := os.Getenv("CALENDAR_ALLOWED_ORIGINS"); envOrigins != "" {
		*allowedOrigins = envOrigins
	}

	//	Auth email
	if envAuthEmail := os.Getenv("CALENDAR_AUTHEMAIL"); envAuthEmail != "" {
		*authEmail = envAuthEmail
	}

	//	Auth subject
	if envAuthSubject := os.Getenv("CALENDAR_AUTHSUBJECT"); envAuthSubject != "" {
		*authSubject = envAuthSubject
	}

}

func main() {
	//	Parse environment variables:
	parseEnvironment()

	//	Parse the command line for flags:
	flag.Parse()

	name := "projects/329817979379/secrets/credentials/versions/3"
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Fatal(err)
	}
	temp := string(result.Payload.Data)

	jsonData := []byte(temp)

	type Cred struct {
		Type             string
		ProjectID        string
		PrivateKeyID     string
		PrivateKey       string
		ClientEmail      string
		ClientID         string
		AuthURI          string
		TokenURI         string
		AuthProviderCert string
		ClientCert       string
	}

	var out Cred
	err2 := json.Unmarshal(jsonData, &out)
	if err != nil {
		log.Println(err2)
	}

	// Your credentials should be obtained from the Google
	// Developer Console (https://console.developers.google.com).
	conf := &jwt.Config{
		Email: out.ClientEmail,

		PrivateKey: []byte(out.PrivateKey),
		Scopes: []string{
			calendar.CalendarScope,
			calendar.CalendarReadonlyScope,
		},
		TokenURL: google.JWTTokenURL,

		Subject: *authSubject,
	}

	client2 := conf.Client(oauth2.NoContext)

	r := mux.NewRouter()
	r.HandleFunc("/calendar/{calendarid}", func(w http.ResponseWriter, r *http.Request) {

		//	Parse the calendarid from the url
		id := mux.Vars(r)["calendarid"]

		//	Get a connection to the calendar service
		//	If we have errors, return them using standard HTTP service method
		svc, err := calendar.New(client2)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//	Get the list of events from now until the end of today
		now := time.Now()
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Add(24 * time.Hour)

		events, err := svc.Events.List(id).
			TimeMin(now.Format(time.RFC3339)).
			TimeMax(end.Format(time.RFC3339)).
			SingleEvents(true).
			OrderBy("startTime").Do()

		//	If we have errors, return them using standard HTTP service method
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//	Set the content type header and return the JSON
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(events)
	})

	//	CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(*allowedOrigins, ","),
		AllowCredentials: true,
	})
	handler := c.Handler(r)

	//	Indicate what port we're starting the service on
	portString := strconv.Itoa(*port)
	fmt.Println("Allowed origins: ", *allowedOrigins)
	fmt.Println("Starting server on :", portString)
	fmt.Println(out.PrivateKey)
	http.ListenAndServe(":"+portString, handler)
}
