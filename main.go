package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

var (
	db  *sql.DB
	err error
)

func main() {
	fmt.Println("Starting Reservation API")

	// Configuratiebestand initialiseren
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetDefault("dbusername", "root")
	viper.SetDefault("dbpassword", "")
	viper.SetDefault("dbaddress", "localhost")
	viper.SetDefault("dbport", 3306)
	viper.SetDefault("dbname", "test")
	viper.SetDefault("dbtable", "users")
	viper.SetDefault("httpport", 15000)

	// Leest config file uit
	fmt.Println("Configuratiebestand lezen..")
	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config-bestand is niet gevonden, er wordt een nieuw bestand gemaakt.")
			err = viper.SafeWriteConfig()
			if err != nil {
				fmt.Printf("Fout bij maken van configuratiebestand: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Wijzig de instelling in het bestand en start het programma opnieuw")
			os.Exit(1)
		} else {
			fmt.Printf("Fout bij het lezen van het configuratiebestand: %v\n", err)
		}
	}

	// get variables from config
	username := viper.GetString("dbusername")
	password := viper.GetString("dbpassword")
	address := viper.GetString("dbaddress")
	port := viper.GetInt("dbport")
	dbName := viper.GetString("dbname")

	// connect to database
	fmt.Printf("Verbinding maken met MySQL-server: %s:%d\n", address, port)
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", username, password, address, port, dbName))
	if err != nil {
		fmt.Printf("Fout bij  het opstarten van mysql-server:%v\n", err)
		os.Exit(1)
	}
	fmt.Println("MySQL-server succesvol verbonden")

	fmt.Println("http api-server start")
	// setup http server
	http.HandleFunc("/create", handleDataReceived)

	// start listening http request
	err := http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("httpport")), nil)
	if err != nil {
		fmt.Printf("Fout bij luisteren http-verzoek: %v\n", err)
		os.Exit(1)
	}
}

// Data struct from ESPHome
type Data struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// handleDataReceivedRequest
func handleDataReceived(w http.ResponseWriter, r *http.Request) {
	// retourneer als verzoektype niet is geplaatst en stuur http-status 500 (intern server error)
	if r.Method != "POST" {
		_, err := fmt.Fprintln(w, "API kan alleen worden aangeroepen door POST Request")
		if err != nil {
			fmt.Printf("Schrijven naar http-clientfout: %v\n", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// lees de volledige aanvraagtekst
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Fout bij het lezen van HTTP-aanvraagtekst: %v\n", err)
		return
	}

	// gegevens ophalen uit de body
	data := Data{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Printf("Unmarshal json van HTTP-aanvraag body-fout: %v\n", err)
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// bereid mysql-query voor
	stmt, err := db.Prepare("INSERT INTO " + viper.GetString("dbtable") + " (name,email) VALUES(?,?);")
	if err != nil {
		fmt.Printf("Mysql-queryfout voorbereiden: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// mysql-query uitvoeren
	_, err = stmt.Exec(data.Name, data.Email)
	if err != nil {
		fmt.Printf("Uitvoeren van mysql-queryfout: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "Success")
}
