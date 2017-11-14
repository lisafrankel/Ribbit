package main

import (
	"net/http"
	"fmt"
	"html/template"
	"time"
	"encoding/json"
	"net/url"
	"log"
)

type post struct {
	Username string
	Post_data string
	Time_id int
}

// handler for main feed page, to post and see other's posts
func post_croak(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 301)
		return
	} else if r.Method == http.MethodPost {
		r.ParseForm()
		resp, _ := http.PostForm("http://localhost:8081/main", url.Values{"post_data": {r.PostFormValue("post")},"username": {userCookie.Value}})
		if resp.StatusCode == 512 {
			fmt.Fprintf(w, "post greater than 100 characters") //how do receive specific error from server?
			return
		}
	}

	// response here is []posts - userfeed // this changes to /getFeed
	response, err := http.PostForm("http://localhost:8081/getUserFeed", url.Values {"username": {userCookie.Value}})
	if err != nil {
		log.Fatal(err)
	}
	
	dec := json.NewDecoder(response.Body)
	var user_feed []post
	dec_err := dec.Decode(&user_feed)
	
	if dec_err != nil {
		log.Fatal(dec_err)
	}

	t := template.New("some_name")
	t, _ = template.ParseFiles("main_feed.html")
	t.Execute(w, user_feed)
}


// login handler
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles("ribbit_home_page.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		resp, _ := http.PostForm("http://localhost:8081/login", url.Values{"username": {r.PostFormValue("username")}, "password":{r.PostFormValue("password")}})
		if resp.StatusCode == 601 {
			fmt.Fprintf(w, "User does not exist")
		} else if resp.StatusCode == 602 {
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
	}
	username := userCookie.Value
	if r.Method == "GET" {
		t, _ := template.ParseFiles("manage_friends.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		friend := r.PostFormValue("username")
		resp, _ := http.PostForm("http://localhost:8081/manage", url.Values{"username": {username}, "friend": {friend}, "choice": {r.Form.Get("choice")}})
		if resp.StatusCode == 605 {
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
		
		http.PostForm("http://localhost:8081/delete", url.Values{"username": {username}})

		http.Redirect(w, r, "/login", 301)
	}
}


// handler for logging out of site
func logout(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		http.Redirect(w, r, "/login", 301)
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
	if r.Method == "GET"  {
		t, _ := template.ParseFiles("create_account.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		resp, _ := http.PostForm("http://localhost:8081/create", url.Values{"username": {r.PostFormValue("username")}, "name": {r.PostFormValue("name")}, "email": {r.PostFormValue("email")},"password": {r.PostFormValue("password")}})
		if resp.StatusCode == 607 {
			fmt.Fprintf(w, "Username already exists")
		} else if resp.StatusCode == 606 {
			fmt.Fprintf(w, "Invalid entry")
		} else {
			http.Redirect(w, r, "/login", 301)
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
