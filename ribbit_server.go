package main

import (
	"io/ioutil"
	"net/http"
	"log"
	"fmt"
	"html/template"
	"time"
)

// struct for a user object, where user data is stored ... (friend_list is map of all their friends by username)
type user struct {
	username string
	password string
	email string
	name string
	friend_list map[string]bool
}

// struct for a post object, where post data is stored 
type post struct {
	Username string
	Post_data string
	Time_id int
}

// global data, Users is a map of Users & their data, Posts is a map of posts & associate data, time_track used a happened-before time_id for posts
var Users = make(map[string]user)
var Posts []post
var time_track int = 0

// function for selecting posts for users based on their friends list
func createUserFeed(username string) []post {
	var user_feed []post
	for k, _ := range Users[username].friend_list {
		for i := 0; i < len(Posts); i++ {
			if Posts[i].Username == k {
				user_feed = append(user_feed, Posts[i])
			}
		}
	}
	return user_feed
}

// handler for main feed page, to post and see other's posts
func post_croak(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		form, err := ioutil.ReadFile("ribbit_home_page.html")
		if err != nil {
				log.Fatal(err)
			}
		fmt.Fprintf(w, string(form))
	} else {
		switch r.Method {
			case http.MethodPost:
				r.ParseForm()
				if len(r.PostFormValue("post")) > 100 {
					fmt.Fprintf(w, "Post greater than 100 characters not allowed")
				} else {

					// add users post to Post list
					var userPost post
					userPost.Post_data = r.PostFormValue("post")
					userPost.Time_id = time_track
					userPost.Username = userCookie.Value
					time_track++
					Posts = append(Posts, userPost)
				}
		}

		// create custom feed for user based on friends & template it
		user_feed := createUserFeed(userCookie.Value)
		t := template.New("some_name")
		t, _ = template.ParseFiles("main_feed.html")
		t.Execute(w, user_feed)
	}
}


// checks to see if user exists in Users map
func userExists(username string) bool {
	_, ok := Users[username]
	return ok
}

// checks to see if password correct for the username
func passwordCorrect(username string, password string) bool {
	if (Users[username].password == password) {
		return true
	}
	return false
}

// login handler
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles("ribbit_home_page.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")
		//fmt.Println("username: ", username)
		//fmt.Println("password: ", password)
		
		// make sure username is associated with an accpunt
		if !userExists(username) {
			fmt.Fprintf(w, "No account exists.")

		// make sure password is correct
		} else if !passwordCorrect(username, password) {
			fmt.Fprintf(w, "Wrong Password")
		} else {

			// create cookie for user, and log them in by redirecting to /main
			cookieValue := username
			expire := time.Now().Add(1 * time.Hour)
			userCookie := http.Cookie{Name: "SessionID", Value: cookieValue, Expires: expire}
			http.SetCookie(w, &userCookie)
			http.Redirect(w, r, "/main", 301)
		}
	}
}

// add friend to a users friend list
func addFriendForUser(username string, friend string) {
	Users[username].friend_list[friend] = true
}

// remove friend from a users friend list
func removeFriendForUser(username string, friend string) {
	delete(Users[username].friend_list, friend)
}

// check whether 'friend' is in 'usernames' friend list
func areFriends(username string, friend string) bool {
	_, ok := Users[username].friend_list[friend]
	return ok
}

// handler for adding and removing friends
func manageFriends(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		form, err := ioutil.ReadFile("ribbit_home_page.html")
		if err != nil {
				log.Fatal(err)
			}
		fmt.Fprintf(w, string(form))
	}
	username := userCookie.Value
	if r.Method == "GET" {
		t, _ := template.ParseFiles("manage_friends.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		friend := r.PostFormValue("username")
		if userExists(friend) {
			choice := r.Form.Get("choice") // are they adding or removing friend
			if choice == "add" { 
				if areFriends(username, friend) { // make sure they are not already friends
					fmt.Fprintf(w, "user %s is already your friend!", friend)
				} else if friend == username { // can't delete yourself as a friend
					fmt.Fprintf(w, "%s if you!", friend)
				} else  {
					addFriendForUser(username, friend)
				}
			} else { // choice is to remove friend
				if !areFriends(username, friend) {
					fmt.Fprintf(w, "user %s was never your friend!", friend)
				} else {
					removeFriendForUser(username, friend)
				}
			}
			http.Redirect(w, r, "/manage", 301)
		} else {
			fmt.Fprintf(w, "user %s does not exist", friend)
		}
	}
}


// handler for deleting a user from the site
func deleteUser(username string) {

	// delete user from Users "database"
	delete(Users, username)

	// delete all posts the user made
	for i := 0; i < len(Posts); i++ {
		if Posts[i].Username == username {
			Posts[len(Posts) - 1], Posts[i] = Posts[i], Posts[len(Posts) - 1]
			Posts = Posts[:len(Posts) - 1]
		}
	}

	// delete user from other user's friend lists
	for k1, _ := range Users {
		for k2, _ := range Users[k1].friend_list {
			if k2 == username {
				delete(Users[k1].friend_list, k2)
			}

		}
	}
}

// handler for deleting user account
func deleteAccount(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		form, err := ioutil.ReadFile("ribbit_home_page.html")
		if err != nil {
				log.Fatal(err)
			}
		fmt.Fprintf(w, string(form))
	}
	//fmt.Println("username from cookie is: ", userCookie.Value)
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
		deleteUser(username)
		http.Redirect(w, r, "/login", 301)
	}
}


// handler for logging out of site
func logout(w http.ResponseWriter, r *http.Request) {
	userCookie, _ := r.Cookie("SessionID")
	if userCookie == nil {
		form, err := ioutil.ReadFile("ribbit_home_page.html")
		if err != nil {
				log.Fatal(err)
			}
		fmt.Fprintf(w, string(form))
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
	if r.Method == "GET" {
		t, _ := template.ParseFiles("create_account.html")
		t.Execute(w, nil)
	} else if r.Method == "POST" {
		r.ParseForm()
		username := r.PostFormValue("username")
		name := r.PostFormValue("name")
		email := r.PostFormValue("email")
		password := r.PostFormValue("password")
		
		// make sure all fields have an input value, not an empty string
		if (username == "" || name == "" || email == "" || password == "") {
			fmt.Fprintf(w, "you did not fill out form correctly")
		} else {

			// add new user to map Users, so long as the user does not already exist
			var new_user user
			new_user.name = name
			new_user.username = username
			new_user.password = password
			new_user.email = email
			new_user.friend_list = make(map[string]bool)
			new_user.friend_list[username] = true
			if userExists(r.PostFormValue("username")) {
				fmt.Fprintf(w, "Username %s already exists", username)
			} else {
				Users[username] = new_user
				http.Redirect(w, r, "/login", 301)
			}
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
