# Uphold Assessment

The price tracker has the ability to alert the user of any price oscillations to the ask and/or the bid price for a specific currency pair. Alerting is done via logging on the console and using a database.

By default the application tracks the monitoring of the `BTC-USD` currency pair at 5s intervals and will alert if either the bid price or the ask price fluctuates more than 0.01%

The tracker uses mongoDB as a data store. For more information about the database operations, schema design and interfacing mechanisms check below.

I have purposely made the code a bit verbose to improve readability, as there a couple of moving parts.

```
.
├── Dockerfile
├── README.md
├── cmd
│   └── main.go
├── docker-compose.yml
├── go.mod
├── go.sum
├── internal
│   ├── config
│   │   └── config.go
│   ├── db
│   │   ├── db.go
│   │   └── mock_db.go
│   └── tracker
│       ├── record.go
│       ├── tracker.go
│       └── tracker_test.go

5 directories, 12 files

```

## Configuration

The application has an `.env` file defined with default values. The user has the ability to customize the tracker by changing those env variables.

- `CURRENCY_PAIRS` : comma separated values of ISO currency code pairs
- `FETCH_INTERVAL` : time in seconds for tracker to retrieve currency prices
- `OSC_PERCENTAGE` : percentage oscillation threshold. If the price goes in either direction of the threshold, alert workflow will be triggered
- `PRICE`: type of price to monitor, available values - `ASK`, `BID`, `BOTH`

Default values are present in the code and in the `.env` file

## Commands

To start the tracker and the database,

```bash
    go mod vendor && docker-compose up --build -d
```

To stop the service,

```bash
    docker-compose down
```

## Troubleshooting

The application may crash when you run it for the first time because it can't connect to the DB with the following error,

```
error occured during connection handshake: dial tcp 172.19.0.3:27017: connect: connection refused
```

This is because the application and the database start at the same time. I have placed a timeout of 60s, but there have been occasions where it doesn't work. A simple rerun of the above command (or `docker-compose up`) will recreate the application container and since the database is already running, there should not be any issues.

# How it works?

The price tracker is initialized through `ENV` variables. Don't worry if you try to run it locally (without a docker image using `go run`), because the code will automatically populate default values. Once initalized, the code will spawn goroutines depending on the number of currency-pairs you wish to monitor. Each routine will fetch prices at intervals corresponding to the `FETCH_INTERVAL` value before hitting the uphold sandbox API to get the latest prices for the corrsponding currency-pair. It will then check to see if the price has flucutated by a value greater than or equal to the `OSC_PERCENTAGE` and if so it will log on the console as well as upload a record of this fluctuation on the database.

I assumed that the monitoring is always done from a base price, and hence when the threshold has been breached, the latest rate becomes the base value for subsequent oscillations. Hence when we start the service, the first value received by the endpoint will be the base price and a record of this will be stored on the database.

The code doesn't store the latest rate (combined) in it's state, but monitors the for the `ASK` and the `BID` price separately. However, the latest combined rates will be available on the console and on the database when a fluctuation occurs.

The tracker can be configured to listen out for both variations in `ASK` or `BID` price. By default, the code monitors both these values, which is configured by the env variable `PRICE`. By setting this value to a specific price type, you can restrict the tracker to only monitor one or the other.

_I added a `TODO` relating to dynamically formatting decimal places when writing to the database. Since there are no straight-forward solutions for this, I opted against including it in the source code, to maintain readability_

# Database

The service uses a Mongodb to log any price changes. I assumed, from the context given in the problem, that this was a merely a logging mechanism and hence a NoSQL database seemed to fit the requirements. Coupled with the fact that the scope of the assignment did not detail any joins, I chose MongoDB to fulfill this criteria.

The docker-compose file also contains a UI viewer (`mongo-express`) for the database, to easily visualize the data being inserted onto mongo. The UI takes a couple of seconds to boot, but should be viewable at the following URL -
`http://localhost:27018/db/db`.

_Note :- `mongo-express` credentials can be found in the `.env` file_

## Schema

The following is an example document that is inserted into the database,

```json
{
  "time": "ISODate('2021-06-21T11:55:14.406Z')",
  "currencypair": "ETH-USD",
  "rate": {
    "ask": "1976.01608",
    "bid": "1967.31938"
  },
  "diff": {
    "value": 1.1239,
    "percentage": 0.0569
  },
  "settings": {
    "fetchinterval": 5,
    "oscpercentage": 0.01,
    "price": "ask"
  }
}
```

The document contains all relevant information regarding price fluctuations including timestamp, corresponding currency-pair, tracker settings and associated prices at the given time. The document also details the delta in price variation in both absolute value and in percentage. The field `settings.price` will indicate which price value changed.

_Note :- `diff.value` and `diff.percentage` will be negative in case of price decrease_
