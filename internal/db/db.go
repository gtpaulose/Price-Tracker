package db

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const colName string = "prices"

type (
	Storage interface {
		Store(interface{}) error
	}

	DB struct {
		client *mongo.Client
		dbName string
	}
)

func NewDB() *DB {
	var connectionString string = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s",
		viper.GetString("MONGODB_USERNAME"),
		viper.GetString("MONGODB_PASSWORD"),
		viper.GetString("MONGODB_HOST"),
		viper.GetString("MONGODB_PORT"),
		viper.GetString("MONGODB_DATABASE"),
	)

	//Initialize the context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	//Connect to the DB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
	if err != nil {
		log.Fatalln("Connection error", err)
	}

	//Check connection status
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalln("MongoDB not reachable. Terminating.\nError: " + err.Error())
	}

	return &DB{client, viper.GetString("MONGODB_DATABASE")}
}

func (db *DB) Store(object interface{}) error {
	con := db.client
	col := con.Database(db.dbName).Collection(colName)

	_, err := col.InsertOne(context.Background(), object)

	return err
}
