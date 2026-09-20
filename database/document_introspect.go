package database

import (
	"context"
	"fmt"

	"github.com/gocql/gocql"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func IntrospectMongo(ctx context.Context, config Config) (Catalog, error) {
	if !config.ReadOnly {
		return Catalog{}, fmt.Errorf("MongoDB introspection requires NYANSAPO_DB_READ_ONLY=true")
	}
	endpoint, err := config.DocumentURL()
	if err != nil {
		return Catalog{}, err
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(endpoint))
	if err != nil {
		return Catalog{}, fmt.Errorf("connect to MongoDB: %w", err)
	}
	defer client.Disconnect(ctx)
	if err := client.Ping(ctx, nil); err != nil {
		return Catalog{}, fmt.Errorf("ping MongoDB: %w", err)
	}
	database := client.Database(config.Database)
	names, err := database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return Catalog{}, fmt.Errorf("list MongoDB collections: %w", err)
	}
	catalog := Catalog{Provider: ProviderMongoDB, Database: config.Database}
	for _, name := range names {
		collection := database.Collection(name)
		var sample bson.M
		if err := collection.FindOne(ctx, bson.D{}).Decode(&sample); err != nil && err != mongo.ErrNoDocuments {
			return Catalog{}, fmt.Errorf("sample MongoDB collection %s: %w", name, err)
		}
		catalog.Collections = append(catalog.Collections, Collection{Name: name, Fields: bsonFields("", sample)})
	}
	return catalog, nil
}

func bsonFields(prefix string, document bson.M) []Field {
	fields := make([]Field, 0, len(document))
	for name, value := range document {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		typeName := fmt.Sprintf("%T", value)
		fields = append(fields, Field{Path: path, Type: typeName})
		if nested, ok := value.(bson.M); ok {
			fields = append(fields, bsonFields(path, nested)...)
		}
	}
	return fields
}

func IntrospectCassandra(ctx context.Context, config Config) (Catalog, error) {
	if !config.ReadOnly {
		return Catalog{}, fmt.Errorf("Cassandra introspection requires NYANSAPO_DB_READ_ONLY=true")
	}
	cluster := gocql.NewCluster(config.Host)
	cluster.Keyspace = config.Database
	if config.Port > 0 {
		cluster.Port = config.Port
	}
	cluster.Consistency = gocql.One
	if config.User != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{Username: config.User, Password: config.Password}
	}
	session, err := cluster.CreateSession()
	if err != nil {
		return Catalog{}, fmt.Errorf("connect to Cassandra: %w", err)
	}
	defer session.Close()

	catalog := Catalog{Provider: ProviderCassandra, Database: config.Database}
	tableIter := session.Query("SELECT table_name FROM system_schema.tables WHERE keyspace_name = ?", config.Database).WithContext(ctx).Iter()
	var name string
	for tableIter.Scan(&name) {
		table := Table{Name: name, Schema: config.Database}
		iter := session.Query("SELECT column_name, type, kind FROM system_schema.columns WHERE keyspace_name = ? AND table_name = ?", config.Database, name).WithContext(ctx).Iter()
		var columnName, columnType, kind string
		for iter.Scan(&columnName, &columnType, &kind) {
			table.Columns = append(table.Columns, Column{Name: columnName, Type: columnType, PrimaryKey: kind == "partition_key" || kind == "clustering"})
		}
		if err := iter.Close(); err != nil {
			return Catalog{}, fmt.Errorf("read Cassandra columns for %s: %w", name, err)
		}
		catalog.Tables = append(catalog.Tables, table)
	}
	if err := tableIter.Close(); err != nil {
		return Catalog{}, fmt.Errorf("list Cassandra tables: %w", err)
	}
	return catalog, nil
}
