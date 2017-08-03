package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"log"

	"time"

	"github.com/lyft/lyft-go-sdk/lyft"
	"golang.org/x/oauth2"
)

type (
	user struct {
		ID           bson.ObjectId `bson:"_id"`
		LyftID       string        `bson:"lyft_user_id"`
		RefreshToken string        `bson:"refresh_token"`
	}
)

var (
	ctx = context.Background()
	wg  sync.WaitGroup

	config = &oauth2.Config{
		ClientID:     "YOUR_CLIENT_ID",
		ClientSecret: "YOUR_CLIENT_SECRET",
		Scopes:       []string{"public", "profile", "rides.read", "offline"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://api.lyft.com/oauth/authorize",
			TokenURL: "https://api.lyft.com/oauth/token",
		},
	}
	oauthStateString = "random"
)

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/auth", handleAuth)
	http.HandleFunc("/success", handleAuthSuccess)
	http.HandleFunc("/redirect", handleAuthRedirect)
	http.HandleFunc("/poll", handlePoll)

	fmt.Println("Started running on http://127.0.0.1:8000")
	fmt.Println(http.ListenAndServe(":8000", nil))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body><a href="/auth">Login with Lyft</a></body></html>`))
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	url := config.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleAuthRedirect(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		log.Fatalf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("oauthConf.Exchange(): %s\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	httpClient := config.Client(ctx, token)
	client := lyft.NewAPIClient(httpClient, "lyft-loyalty-program")

	result, _, err := client.UserApi.GetProfile()

	if err != nil {
		log.Fatal(err)
	} else {
		session, err := mgo.Dial("localhost")
		if err != nil {
			log.Fatalf("mongo: %s\n", err)
		}
		defer session.Close()
		session.SetMode(mgo.Monotonic, true)

		u := user{
			ID:           bson.NewObjectId(),
			LyftID:       result.Id,
			RefreshToken: token.RefreshToken,
		}

		c := session.DB("store").C("users")
		if err := c.Insert(u); err != nil {
			log.Fatalf("mongo: %s\n", err)
		}

	}

	http.Redirect(w, r, "/success", http.StatusTemporaryRedirect)
	return
}

func handleAuthSuccess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body>Success!</body></html>`))
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Fatalf("mongo: %s\n", err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	var users []user
	err = session.DB("store").C("users").Find(nil).All(&users)
	if err != nil {
		log.Fatalf("mongo: %s\n", err)
	}

	tasks := make(chan user, len(users))

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go getUserRides(tasks)
	}

	for _, user := range users {
		tasks <- user
	}
	close(tasks)

	wg.Wait()
	w.WriteHeader(http.StatusOK)

}

func getUserRides(users chan user) {
	defer wg.Done()

	for {
		user, ok := <-users

		if !ok {
			return
		}

		token := oauth2.Token{
			RefreshToken: user.RefreshToken,
		}

		httpClient := config.Client(ctx, &token)
		client := lyft.NewAPIClient(httpClient, "lyft-loyalty-program")

		previousDay := time.Now().AddDate(0, 0, -1)
		previousDayStart := time.Date(previousDay.Year(), previousDay.Month(), previousDay.Day(), 0, 0, 0, 0, time.UTC)
		previousDayEnd := map[string]interface{}{
			"endTime": time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC),
		}

		resp, _, err := client.UserApi.GetRides(previousDayStart, previousDayEnd)
		if err != nil {
			log.Fatalf("lyft api: %s\n", err)
		}

		for _, ride := range resp.RideHistory {
			if ride.Status == lyft.RideStatusDroppedOff {
				fmt.Printf("Ride ID %v distance: %v miles\n", ride.RideId, ride.DistanceMiles)
				// Calculate Award Points for each Lyft mile
				// ...
			}
		}
	}
}
