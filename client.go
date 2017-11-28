package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
	"net"
	"bytes"
)

type post struct {
	Username  string
	Post_data string
	Time_id   int
}

// handler for main feed page, to post and see other's posts
func post_croak(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 301)
		return
	}
	if r.Method == http.MethodPost {
		r.ParseForm()
		post := r.PostFormValue("post")
		username := userCookie.Value

		conn, _ := net.Dial("tcp", "localhost:8081")
		client_msg := "post," + post + "," + username
		fmt.Fprintf(conn, client_msg)

		response := make([]byte, 1024)
		n, err := conn.Read(response)
		if err != nil {
			fmt.Println(err)
			}
		s_message := string(response[:n])

		if s_message == "512" {
			fmt.Fprintf(w, "post greater than 100 characters")
		}
		conn.Close()
	}

	conn1, _ := net.Dial("tcp", "localhost:8081")
	defer conn1.Close()
	client_msg := "getUserFeed," + userCookie.Value
	fmt.Fprintf(conn1, client_msg)

	response := make([]byte, 1024)
	_, err := conn1.Read(response)
	if err != nil {
		fmt.Println(err)
		}

	reader := bytes.NewReader(response)
	dec := json.NewDecoder(reader)
	var user_feed []post
	dec_err := dec.Decode(&user_feed)

	if dec_err != nil {
		fmt.Println(dec_err)
	}

	t := template.New("some_name")
	t, _ = template.ParseFiles("main_feed.html")
	t.Execute(w, user_feed)
}

// login handler
func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in login func")
	userCookie, _ := r.Cookie("SessionID")
	if userCookie != nil {
		http.Redirect(w, r, "/main", 301)
		return
	}
	if r.Method == "GET" {
		t, _ := template.ParseFiles("ribbit_home_page.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")	


		conn, _ := net.Dial("tcp", "localhost:8081")
		defer conn.Close()
		client_msg := "login," + username +"," + password
		fmt.Fprintf(conn, client_msg)
		
		response := make([]byte, 1024)
		n, err := conn.Read(response)
		if err != nil {
			fmt.Println(err)
			}
		s_message := string(response[:n])

		if s_message == "601" {
			fmt.Fprintf(w, "User does not exist")
		} else if s_message == "602" {
			fmt.Fprintf(w, "Incorrect password")
		} else {
			cookieValue := r.PostFormValue("username")
			expire := time.Now().Add(1 * time.Hour)
			userCookie := http.Cookie{Name: "SessionID", Value: cookieValue, Expires: expire}
			http.SetCookie(w, &userCookie)
			http.Redirect(w, r, "/main", 301)
		}
	}
}

// handler for adding and removing friends
func manageFriends(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 301)
		return
	}
	username := userCookie.Value
	if r.Method == "GET" {
		t, _ := template.ParseFiles("manage_friends.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		friend := r.PostFormValue("username")
		choice := r.Form.Get("choice")

		conn, _ := net.Dial("tcp", "localhost:8081")
		defer conn.Close()
		client_msg := "manage," + username + "," + friend + "," + choice
		fmt.Fprintf(conn, client_msg)
		
		response := make([]byte, 1024)
		n, err := conn.Read(response)
		if err != nil {
			fmt.Println(err)
			}
		s_message := string(response[:n])
		
		if s_message == "601" {
			fmt.Fprintf(w, friend + " does not exist")
		} else {
			http.Redirect(w, r, "/manage", 301)
		}
	}
}

// handler for deleting user account
func deleteAccount(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 301)
		return
	}
	if r.Method == "GET" {
		t, _ := template.ParseFiles("delete_account.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {

		// expire users cookie and remove them from User list, redirect to login page
		username := userCookie.Value
		cookieValue := username
		expire := time.Now()
		userCookie := http.Cookie{Name: "SessionID", Value: cookieValue, Expires: expire}
		http.SetCookie(w, &userCookie)

		// tell server to delete username
		conn, _ := net.Dial("tcp", "localhost:8081")
		defer conn.Close()
		client_msg := "delete," + username
		fmt.Fprintf(conn, client_msg)

		http.Redirect(w, r, "/login", 301)
	}
}

// handler for logging out of site
func logout(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 301)
		return
	}
	if r.Method == "GET" {
		t, _ := template.ParseFiles("logout.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {

		// expire cookie & redirect them back to login page
		username := userCookie.Value
		cookieValue := username
		expire := time.Now()
		userCookie := http.Cookie{Name: "SessionID", Value: cookieValue, Expires: expire}
		http.SetCookie(w, &userCookie)
		http.Redirect(w, r, "/login", 301)
	}
}

// handler for making a new account
func makeAccount(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie != nil {
		http.Redirect(w, r, "/main", 301)
		return
	}
	if r.Method == "GET" {
		t, _ := template.ParseFiles("create_account.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		
		// connect to server & define variables that will be sent to server	
		conn, _ := net.Dial("tcp", "localhost:8081")
		defer conn.Close()
		username := r.PostFormValue("username")
		name := r.PostFormValue("name")
		email := r.PostFormValue("email")
		password := r.PostFormValue("password")
		
		// compose message to client, of request, username, name, email & password
		client_msg := "create," + username + "," + name + "," + email + "," + password
		
		// send message to server
		fmt.Fprintf(conn, client_msg)
		//_, err := bufio.NewReader(conn).ReadString('\n')
		response := make([]byte, 1024)
		n, err := conn.Read(response)
		if err != nil {
			fmt.Println(err)
		}

		s_message := string(response[:n])
		if s_message == "200" {
			http.Redirect(w, r, "/login", 301)
			return
		}

		if s_message == "607" {
			fmt.Fprintf(w, "Username already exists")
		} else if s_message == "606" {
			fmt.Fprintf(w, "Invalid entry")
		} 
	}
}

func main() {
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/main", post_croak)
	http.HandleFunc("/create", makeAccount)
	http.HandleFunc("/login", login)
	http.HandleFunc("/delete", deleteAccount)
	http.HandleFunc("/manage", manageFriends)
	http.ListenAndServe(":8080", nil)
}
