// Author: Lisa Frankel
// Client server


package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
	"net"
	"bytes"
	"strings"
)

type post struct {
	Username  string
	Post_data string
	Time_id   int
}

// check errors
func check(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// function returns which server the client should READ from
func getReadServer() string {
	conn, _ := net.Dial("tcp", "localhost:8084") // load balancer
	defer conn.Close()
	fmt.Fprintf(conn, "read")

	response := make([]byte, 1024)
	n, err := conn.Read(response)
	check(err)
	port := string(response[:n])

	fmt.Println("connecting to server: ", port)

	return port
}

// function returns which server the client should write to (clients write to all active servers)
func getWriteServer(w http.ResponseWriter) []string {
	conn, _ := net.Dial("tcp", "localhost:8084") // load balancer
	defer conn.Close()
	fmt.Fprintf(conn, "write")

	message := make([]byte, 1024)
	n, err := conn.Read(message)
	check(err)

	// format message into tokens
	s_message := string(message[:n])
	fmt.Println(s_message)

	if s_message == "nil,nil,nil" {
		var servers []string
		return servers
	}

	message_tokens := strings.Split(s_message, ",")

	var servers []string
	
	for i := 0; i < len(message_tokens); i++ {
		servers = append(servers, message_tokens[i])
	}

	return servers
}

// tell balancer a server is not responding
func tellBalancer(port string) {
	conn := writeToServer("8084", "down," + port)
	conn.Close()
}

// writes client_msg to the server that listening on port
func writeToServer(port string, client_msg string) net.Conn {
	conn, err := net.Dial("tcp", "localhost:" + port)

	if err != nil {
		tellBalancer(port)
		return nil
	}

	fmt.Fprintf(conn, client_msg)

	return conn

}

// handler for main feed page, to post and see other's posts
func post_croak(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 302)
		return
	}
	if r.Method == http.MethodPost {
		r.ParseForm()
		post := r.PostFormValue("post")
		username := userCookie.Value

		client_msg := "post," + post + "," + username

		// Since posting does a WRITE, send message to all three servers.  However, only read response from 1.
		var conn net.Conn

		servers := getWriteServer(w)
		if len(servers) == 0 {
			fmt.Fprintf(w, "all servers are down")
			return
		}
		for i := 0; i < len(servers); i++ {
			if servers[i] != "nil" { // if server isn't down
				conn1 := writeToServer(servers[i], client_msg)
				if conn1 != nil { // if we discover that server crashed
					conn = conn1
				}
			}
		}


		response := make([]byte, 1024)
		n, err := conn.Read(response)
		check(err)
		s_message := string(response[:n])

		if s_message == "512" {
			fmt.Fprintf(w, "post greater than 100 characters")
		}

		conn.Close()
	}

	// get server, since we are doing a read here
	port := getReadServer()
	conn, err := net.Dial("tcp", "localhost:" + port)
	for err != nil {
		tellBalancer(port)
		port = getReadServer()
		if port == "nil" {
			fmt.Fprintf(w, "All servers are down")
			return
		}
		conn, err = net.Dial("tcp", "localhost:" + port)
	}

	defer conn.Close()
	client_msg := "getUserFeed," + userCookie.Value
	fmt.Fprintf(conn, client_msg)

	response := make([]byte, 1024)
	_, err1 := conn.Read(response)
	check(err1)

	reader := bytes.NewReader(response)
	dec := json.NewDecoder(reader)
	var user_feed []post
	dec_err := dec.Decode(&user_feed)

	check(dec_err)

	t := template.New("some_name")
	t, _ = template.ParseFiles("main_feed.html")
	t.Execute(w, user_feed)
}

// login handler
func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("in login func")
	userCookie, _ := r.Cookie("SessionID")
	if userCookie != nil {
		http.Redirect(w, r, "/main", 302)
		return
	}
	if r.Method == "GET" {
		t, _ := template.ParseFiles("ribbit_home_page.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")	

		if username == "" || password == "" {
			fmt.Fprintf(w, "Either username or password field blank")
			http.Redirect(w, r, "/main", 302)
			return
		}

		// use getReadServer since we are only doing a read here
		port := getReadServer()
		conn, err := net.Dial("tcp", "localhost:" + port)
		for err != nil {
			tellBalancer(port)
			port = getReadServer()
			if port == "nil" {
				fmt.Fprintf(w, "All servers are down")
				return
			}
			conn, err = net.Dial("tcp", "localhost:" + port)
		}	
		defer conn.Close()
		client_msg := "login," + username +"," + password
		fmt.Fprintf(conn, client_msg)
		
		response := make([]byte, 1024)
		n, err := conn.Read(response)
		check(err)
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
			http.Redirect(w, r, "/main", 302)
		}
	}
}

// handler for adding and removing friends
func manageFriends(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 302)
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

		client_msg := "manage," + username + "," + friend + "," + choice

		var conn net.Conn

		servers := getWriteServer(w)
		if len(servers) == 0 {
			fmt.Fprintf(w, "all servers are down")
			return
		}
		for i := 0; i < len(servers); i++ {
			if servers[i] != "nil" {
				conn1 := writeToServer(servers[i], client_msg)
				if conn1 != nil { // if that server crashed
					conn = conn1
				}
			}
		}
		
		response := make([]byte, 1024)
		n, err := conn.Read(response)
		check(err)
		s_message := string(response[:n])
		
		if s_message == "601" {
			fmt.Fprintf(w, friend + " does not exist")
		} else {
			http.Redirect(w, r, "/manage", 302)
		}
	}
}

// handler for deleting user account
func deleteAccount(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 302)
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
		// send message to all 3 servers since we are doing a write

		client_msg := "delete," + username

		var conn net.Conn

		servers := getWriteServer(w)
		if len(servers) == 0 {
			fmt.Fprintf(w, "all servers are down")
			return
		}
		for i := 0; i < len(servers); i++ {
			if servers[i] != "nil" { // if server isn't down
				conn1 := writeToServer(servers[i], client_msg)
				if conn1 != nil { // if we discover that server crashed
					conn = conn1
				}
			}
		}

		http.Redirect(w, r, "/login", 302)
		conn.Close()
	}
}

// handler for logging out of site
func logout(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 302)
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
		http.Redirect(w, r, "/login", 302)
	}
}

// handler for making a new account
func makeAccount(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie != nil {
		http.Redirect(w, r, "/main", 302)
		return
	}
	if r.Method == "GET" {
		t, _ := template.ParseFiles("create_account.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		
		username := r.PostFormValue("username")
		name := r.PostFormValue("name")
		email := r.PostFormValue("email")
		password := r.PostFormValue("password")
		
		// compose message to client, of request, username, name, email & password
		client_msg := "create," + username + "," + name + "," + email + "," + password
		
		// Connect to all servers to do the write
		var conn net.Conn

		servers := getWriteServer(w)
		if len(servers) == 0 {
			fmt.Fprintf(w, "all servers are down")
			return
		}
		for i := 0; i < len(servers); i++ {
			if servers[i] != "nil" { // if server isn't down
				conn1 := writeToServer(servers[i], client_msg)
				if conn1 != nil { // if we discover that server crashed
					conn = conn1
				}
			}
		}


		// send message to server
		response := make([]byte, 1024)
		n, err := conn.Read(response)
		check(err)

		s_message := string(response[:n])
		if s_message == "200" {
			http.Redirect(w, r, "/login", 302)
			return
		}

		if s_message == "607" {
			fmt.Fprintf(w, "Username already exists")
		} else if s_message == "606" {
			fmt.Fprintf(w, "Invalid entry")
		} 

		conn.Close()
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
