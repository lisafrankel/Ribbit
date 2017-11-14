package main

import (
	"io/ioutil"
	"net/http"
	"log"
	"fmt"
	"encoding/json"
	"os"
	"path/filepath"
	"github.com/nu7hatch/gouuid"

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

// function to create user directory, with personal data file, and friends directory
func createUserFile(username string, name string, password string, email string) {
	os.Mkdir("Users/" + username, 0777)
	os.Mkdir("Users/" + username + "/friends", 0777)
	os.Mkdir("Users/" + username + "/posts", 0777)


	// make user friends with self
	self_file, self_err := os.Create("Users/" + username + "/friends/" + username + ".txt")
	defer self_file.Close()

	if self_err != nil {
		log.Fatal(self_err)
	}

	// make file with name
	name_file, name_err := os.Create("Users/" + username + "/name.txt")
	defer name_file.Close()
	if name_err != nil {
		log.Fatal(name_err)
	}

	name_file.WriteString(name)
	
	// make file with password
	password_file, password_err := os.Create("Users/" + username + "/password.txt")
	defer password_file.Close()
	if password_err != nil {
		log.Fatal(password_err)
	}

	password_file.WriteString(password)

	// make file with email
	email_file, email_err := os.Create("Users/" + username + "/email.txt")
	defer email_file.Close()
	if email_err != nil {
		log.Fatal(email_err)
	}
	
	email_file.WriteString(email)

}

// function to delete user file
func deleteUserFile(username string) {
	err := os.RemoveAll("Users/" + username)
	if err != nil {
		log.Fatal(err)
	}
}

// function adds a file with friend's name into user's friend directory
func addFriendForUser(username string, friend string) {
	file, err := os.Create("Users/" + username + "/friends/" + friend + ".txt")
	defer file.Close()

	if err != nil {
		log.Fatal(err)
	}
}

// function removes a file with friend's name from user's friend directory
func removeFriendForUser(username string, friend string) {
	err := os.Remove("Users/" + username + "/friends/" + friend + ".txt")
	if err != nil {
		log.Fatal(err)
	}
}

func userExists(username string) bool {
	if _, err := os.Stat("Users/" + username); os.IsNotExist(err) {
		return false
	}
	return true
}


// checks to see if password correct for the username
func passwordCorrect(username string, password string) bool {
	file, err := os.Open("Users/" + username + "/password.txt")
	defer file.Close()

	if err != nil {
		log.Fatal(err)
	}

	realPassword, _ := ioutil.ReadFile("Users/" + username + "/password.txt")
	if string(realPassword) == password {
		return true
	}
	return false
}


//add post to user's file
func addPostFile(username string, post string) {
	my_uuid, _ := uuid.NewV4()
	file, err := os.Create("Users/" + username + "/posts/" + my_uuid.String() + ".txt")
	defer file.Close()

	if err != nil {
		log.Fatal(err)
	}

	file.WriteString(post)

}


// func to check if users are friends
func areFriends(username string, friend string) bool {
	if _, err := os.Stat("Users/" + username + "/friends/" + friend + ".txt"); os.IsNotExist(err) {
		return false
	}
	return true
}

// function for selecting posts for users based on their friends list
func createUserFeed(username string) []post {
	var user_feed []post
	friends, _ := filepath.Glob("Users/" + username + "/friends/*")
	fmt.Println("friend list", friends)
	for _, friend := range friends {
		friend_name := friend[len("Users/" + username + "/friends/"):len(friend) - 4]
		fmt.Println("friends name", friend_name)
		friend_posts, _ := filepath.Glob("Users/" + friend_name + "/posts/*")
		fmt.Println("friends posts", friend_posts)
		for _, fpost := range friend_posts {
			fmt.Println("file name", fpost)

			// for mac OS, have to add this if statement to prevent opening .DS_Store file
			if fpost != "Users/" + friend_name + "/posts/.DS_Store" {
				new_post_string, _ := ioutil.ReadFile(fpost)
				fmt.Println("new string", new_post_string)
				var new_post post
				new_post.Post_data = string(new_post_string[:])
				new_post.Username = friend_name
				new_post.Time_id = 0
				user_feed = append(user_feed, new_post)
			}
		}
	}
	return user_feed
}



// handler for main feed page, to post and see other's posts
func post_croak(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		fmt.Println("post req main")
		r.ParseForm()
		if len(r.PostFormValue("post_data")) > 100 {
			http.Error(w, http.StatusText(http.StatusInternalServerError), 512)	
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)

		} else {
			
			addPostFile(r.PostFormValue("username"), r.PostFormValue("post_data"))

		}
	}
}

func getUserFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		user_feed := createUserFeed(r.PostFormValue("username"))
		jsonFeed, _ := json.Marshal(user_feed)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonFeed) 
	}
}


// login handler
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		fmt.Println("post req login")
		r.ParseForm()
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")
		
		// make sure username is associated with an accpunt
		if !userExists(username) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), 601)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)

		// make sure password is correct
		} else if !passwordCorrect(username, password) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), 602)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("password inc")
		 }
	}
}


// handler for adding and removing friends
func manageFriends(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		fmt.Println("post req manage")
		r.ParseForm()
		username := r.PostFormValue("username")
		friend := r.PostFormValue("friend")
		if userExists(friend) {
			choice := r.Form.Get("choice") // are they adding or removing friend
			if choice == "add" && !areFriends(username, friend) && friend != username { 
				addFriendForUser(username, friend)
			} else if choice == "remove" && areFriends(username, friend) && friend != username { // choice is to remove friend
				removeFriendForUser(username, friend)
			}
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), 605)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}


// handler for deleting a user from the site
func deleteUser(username string) {


	// delete user from Users "database"
	deleteUserFile(username)


	// delete user from other user's friend lists
	users, _ := filepath.Glob("Users/*")
	for _, user := range users {

		// for mac OS, have to add this if statement to prevent opening .DS_Store file
		if user != "Users/.DS_Store/friends/l.txt" {
			fmt.Println("user: ", user)
			fmt.Println("USER PATH: ", user + "/friends/" + username + ".txt")
			if _, err := os.Stat(user + "/friends/" + username + ".txt"); os.IsNotExist(err) {
				break
			} else {
				e := os.RemoveAll(user + "/friends/" + username + ".txt")
				if e != nil {
					log.Fatal(e)
				}
			}
		}
	}
}


// handler for deleting user account
func deleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		username := r.PostFormValue("username")
		deleteUser(username)
		fmt.Println("delete", r.PostFormValue("username"))
	}
}



// handler for making a new account
func makeAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		username := r.PostFormValue("username")
		name := r.PostFormValue("name")
		email := r.PostFormValue("email")
		password := r.PostFormValue("password")

		fmt.Println("Username: ", username)
		
		// make sure all fields have an input value, not an empty string
		if (username == "" || name == "" || email == "" || password == "") {
			http.Error(w, http.StatusText(http.StatusInternalServerError), 606)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
		} else {

			// add new user to map Users, so long as the user does not already exist
			var new_user user
			new_user.name = name
			new_user.username = username
			new_user.password = password
			new_user.email = email
			new_user.friend_list = make(map[string]bool)
			new_user.friend_list[username] = true
			if userExists(username) {
				http.Error(w, "UserNameExists", 607)
			} else {
				createUserFile(username, name, password, email)
				
			}
		}
	}
}

func main() {
	http.HandleFunc("/main", post_croak)
	http.HandleFunc("/getUserFeed", getUserFeed)
	http.HandleFunc("/create", makeAccount)
    http.HandleFunc("/login", login)
    http.HandleFunc("/delete", deleteAccount)
    http.HandleFunc("/manage", manageFriends)
    http.ListenAndServe(":8081", nil)
}
