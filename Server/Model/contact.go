package Model

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

type UserDb struct {
	host     string
	port     int
	user     string
	password string
	dbname   string
}

var user = AddUser("db_user.csv")

func AddUser(fileName string) UserDb {
	f, err1 := os.Open(fileName)
	if err1 != nil {
		log.Fatal(err1)
	}

	reader := csv.NewReader(f)
	CSVfile, err := reader.ReadAll()

	u := UserDb{}
	u.host = CSVfile[0][0]
	u.port, err = strconv.Atoi(CSVfile[0][1])
	if err != nil {
		panic(err)
	}
	u.user = CSVfile[0][2]
	u.password = CSVfile[0][3]
	u.dbname = CSVfile[0][4]
	return u
}

type Contact struct {
	Id      uint `gorm:"primary key;autoIncrement"`
	Name    string
	Surname string
	Number  string
}

var Contacts = make([]Contact, 0)

var db *gorm.DB

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

func emptyInputDecoder(*http.Request) (interface{}, error) {
	return nil, nil
}

func getNominativeDecoder(r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	key := vars["name"]
	key = fmt.Sprintf(key, vars["surname"])

	return key, nil
}

func getNameDecoder(r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	key := vars["name"]

	return key, nil
}

func getContactDecoder(r *http.Request) (interface{}, error) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var contact Contact
	err = json.Unmarshal(reqBody, &contact)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return contact, nil
}

type Msg struct {
	Msg string
}

func Wrapper(fn func(interface{}) (interface{}, error), dec func(*http.Request) (interface{}, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		payload, err := dec(r)
		if err != nil {
			err = json.NewEncoder(w).Encode(Msg{
				Msg: err.Error(),
			})
			if err != nil {
				return
			}
			return
		}

		resp, err := fn(payload)
		if err != nil {
			err = json.NewEncoder(w).Encode(Msg{
				Msg: err.Error(),
			})
			if err != nil {
				return
			}
			return
		}

		jsonData, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = db.Find(&Contacts).Error
		CheckError(err)
		fmt.Println(Contacts)
		_, err = w.Write(jsonData)
		if err != nil {
			return
		}
	}
}

func homePage(_ interface{}) (interface{}, error) {
	fmt.Println("HomePage!!")
	return nil, nil
}

func allContacts(_ interface{}) (interface{}, error) {
	db.Find(&Contacts)
	return Contacts, nil
}

func getContactByNominative(key interface{}) (interface{}, error) {
	db.Find(&Contacts)
	for _, contact := range Contacts {
		if fmt.Sprintf(contact.Name, contact.Surname) == key {
			return contact, nil
		}
	}
	return nil, fmt.Errorf("nominative not found")
}

func getContactByName(key interface{}) (interface{}, error) {
	db.Find(&Contacts)
	var Fcontact []Contact
	flag := false
	for _, contact := range Contacts {
		if contact.Name == key {
			Fcontact = append(Fcontact, contact)
			flag = true
		}
	}
	if flag {
		return Fcontact, nil
	} else {
		for _, contact := range Contacts {
			if contact.Surname == key {
				Fcontact = append(Fcontact, contact)
				flag = true
			}
		}
		if flag {
			return Fcontact, nil
		} else {
			return nil, fmt.Errorf("name not found")
		}

	}
}

func createNewContact(req interface{}) (interface{}, error) {
	db.Find(&Contacts)
	contact, ok := req.(Contact)
	if !ok {
		return nil, fmt.Errorf("cannot ricognised contact")
	}
	db.Create(&contact)
	fmt.Println(contact)
	Contacts = append(Contacts, contact)
	return contact, nil

}

func deleteContacts(key interface{}) (interface{}, error) {
	db.Find(&Contacts)
	flag := true
	for i, contact := range Contacts {
		if fmt.Sprintf(contact.Name, contact.Surname) == key {

			db.Delete(&Contact{}, contact)
			Contacts = append(Contacts[:i], Contacts[i+1:]...)

			flag = false
		}
	}
	if flag {
		return nil, fmt.Errorf("nominative not found")
	}

	return Contacts, nil
}

func HandleRequests() {
	// connection string
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", user.host, user.port, user.user, user.password, user.dbname)

	// open database
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	CheckError(err)

	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", Wrapper(homePage, emptyInputDecoder))
	myRouter.HandleFunc("/contacts", Wrapper(allContacts, emptyInputDecoder)).Methods("GET")
	myRouter.HandleFunc("/contacts/{name}/{surname}", Wrapper(getContactByNominative, getNominativeDecoder)).Methods("GET")
	myRouter.HandleFunc("/contacts/{name}", Wrapper(getContactByName, getNameDecoder)).Methods("GET")
	myRouter.HandleFunc("/contacts", Wrapper(createNewContact, getContactDecoder)).Methods("POST")
	myRouter.HandleFunc("/contacts/{name}/{surname}", Wrapper(deleteContacts, getNominativeDecoder)).Methods("DELETE")

	http.Handle("/", myRouter)
}
