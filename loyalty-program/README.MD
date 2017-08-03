# Loyalty Program Recipe

Reward your customers for their continued loyalty, whether you're an airline, rental car dealer, hotel, or retailer. Keep track of a customer's Lyft miles and give them points as a reward. This recipe uses our [Go SDK](https://github.com/lyft/Lyft-go-sdk) to poll the API on a regular cadence.

## Registration
- Create a Lyft Developer [account](https://www.lyft.com/developers).
- Once registered, create a new Lyft [application](https://www.lyft.com/developers/manage).
- Your new application will be assigned a Client ID and a Client Secret; write down these values. 
- Set your application's Redirect URL to `http://localhost:8000/redirect`. This is the callback URL where the user will be redirected after successfully authenticating with Lyft. 

## Installation

```console
go get github.com/lyft/lyft-go-samples/loyalty-program
```

## Setup 

1. Replace the default values of `ClientID` and `ClientSecret` parameters by those obtained during Registration. 

2. The recipe uses MongoDB to store the passengers' User Ids and OAuth2 refresh tokens. The latter are used to request valid access tokens, which in turn are used to call the API. The easiest way to install MongoDB is via [brew](https://brew.sh/):

```console
brew install mongodb
```

To start a MongoDB instance, navigate to the `/loyalty-program` directory and run the following command:

```console
mongod --dbpath /db
```

## Usage

The recipes is an HTTP web server that exposes 5 endpoints. All endpoints, except for `/poll`, are used as part of the standard 3-Legged OAuth2 [authentication](https://developer.lyft.com/docs/authentication#section-3-legged-flow-for-accessing-user-specific-endpoints) flow. Run the app and point your browser to `http://localhost:8000/`.

The `/poll` endpoint uses the authenticated users' refresh tokens to ultimately retrieve their previous day's ride history. It should be called on a daily cadency using a cron job. For example, you can leverage Google Cloud Engine [Cron service](https://cloud.google.com/solutions/reliable-task-scheduling-compute-engine) for this task.


