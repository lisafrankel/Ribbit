package main

import (
	"encoding/json"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"net"
	"strings"
	"sync"
)

var filelocks = make(map[string]*sync.RWMutex)


// struct for a post object, where post data is stored
type post struct {
	Username  string
	Post_data string
	Time_id   int
}

func getLock(filename string) {
	if _, ok := filelocks[filename]; ok {
		filelocks[filename].Lock()
		fmt.Println("locking: ", filename, "in getLock")
	} else {
		filelocks[filename] = &sync.RWMutex{}
		fmt.Println("new lock, locking: ", filename, "in getLock")
		filelocks[filename].Lock()
	}
}

func getReadLock(filename string) {
	if _, ok := filelocks[filename]; ok {
		fmt.Println("locking: ", filename, "in getReadLock")
		filelocks[filename].RLock()
	} else {
		filelocks[filename] = &sync.RWMutex{}
		fmt.Println("new lock, locking: ", filename, "in getReadLock")
		filelocks[filename].RLock()
	}
}


// function to create user directory, with personal data file, and friends directory
func createUserFile(username string, name string, password string, email string) {
	os.Mkdir("Users/"+username, 0777)
	os.Mkdir("Users/"+username+"/friends", 0777)
	os.Mkdir("Users/"+username+"/posts", 0777)

	filelocks["Users/"+username] = &sync.RWMutex{}
	fmt.Println("creating lock: ", "Users/", username, "in create user file")

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
	getLock("Users/" + username)
	err := os.RemoveAll("Users/" + username)
	if err != nil {
		log.Fatal(err)
	}
	filelocks["Users/" + username].Unlock()
	fmt.Println("unlocked", "Users/" + username, "in delete user file")
}

// function adds a file with friend's name into user's friend directory
func addFriendForUser(username string, friend string) {
	getReadLock("Users/" + username)
	file, err := os.Create("Users/" + username + "/friends/" + friend + ".txt")
	defer file.Close()
	filelocks["Users/" + username].RUnlock()
	fmt.Println("read unlocking Users/", username)

	if err != nil {
		log.Fatal(err)
	}
}

// function removes a file with friend's name from user's friend directory
func removeFriendForUser(username string, friend string) {
	filename := "Users/" + username + "/friends/" + friend + ".txt"
	getLock(filename)
	getReadLock("Users/" + username)
	err := os.Remove(filename)
	if err != nil {
		log.Fatal(err)
	}
	filelocks[filename].Unlock()
	filelocks["Users/" + username].RUnlock()
	fmt.Println("read unlocking Users/", username)
	fmt.Println("unlocked", filename, "in removeFriendForUser")
}

func userExists(username string) bool {
	getReadLock("Users/" + username)
	if _, err := os.Stat("Users/" + username); os.IsNotExist(err) {
		return false
	}
	filelocks["Users/" + username].RUnlock()
	fmt.Println("read unlocking Users/", username)
	return true
}

// checks to see if password correct for the username
func passwordCorrect(username string, password string) bool {
	filename := "Users/" + username + "/password.txt"
	getReadLock("Users/" + username)
	file, err := os.Open(filename)
	defer file.Close()

	if err != nil {
		log.Fatal(err)
	}

	getReadLock(filename)
	realPassword, _ := ioutil.ReadFile(filename)
	filelocks[filename].RUnlock()
	filelocks["Users/" + username].RUnlock()
	fmt.Println("read unlocking Users/", username)
	fmt.Println("unlocked", filename, "in passwordCorrect")
	if string(realPassword) == password {
		return true
	}
	return false
}

//add post to user's file
func addPostFile(username string, post string) {
	my_uuid, _ := uuid.NewV4()
	filename := "Users/" + username + "/posts/" + my_uuid.String() + ".txt"
	getReadLock("Users/" + username)
	file, err := os.Create(filename)
	defer file.Close()

	if err != nil {
		log.Fatal(err)
	}

	getLock(filename)
	file.WriteString(post)
	
	filelocks[filename].Unlock()
	filelocks["Users/" + username].RUnlock()
	fmt.Println("read unlocking Users/", username)
	fmt.Println("unlocked", filename, "in addPostFile")

}

// func to check if users are friends
func areFriends(username string, friend string) bool {
	getReadLock("Users/" + username)
	if _, err := os.Stat("Users/" + username + "/friends/" + friend + ".txt"); os.IsNotExist(err) {
		return false
	}
	filelocks["Users/" + username].RUnlock()
	fmt.Println("read unlocking Users/", username)
	return true
}


// function for selecting posts for users based on their friends list
func createUserFeed(username string) []post {
	var user_feed []post
	getReadLock("Users/" + username)
	friends, _ := filepath.Glob("Users/" + username + "/friends/*")
	fmt.Println("friend list", friends)
	for _, friend := range friends {
		friend_name := friend[len("Users/"+username+"/friends/") : len(friend)-4]
		fmt.Println("friends name", friend_name)
		getReadLock("Users/"+friend_name)
		friend_posts, _ := filepath.Glob("Users/" + friend_name + "/posts/*")
		fmt.Println("friends posts", friend_posts)
		for _, fpost := range friend_posts {
			fmt.Println("file name", fpost)

			// for mac OS, have to add this if statement to prevent opening .DS_Store file
			if fpost != "Users/"+friend_name+"/posts/.DS_Store" {
				getReadLock(fpost)
				new_post_string, _ := ioutil.ReadFile(fpost)
				filelocks[fpost].RUnlock()
				fmt.Println("unlocked", fpost, "in createUserFeed")
				fmt.Println("new string", new_post_string)
				var new_post post
				new_post.Post_data = string(new_post_string[:])
				new_post.Username = friend_name
				new_post.Time_id = 0
				user_feed = append(user_feed, new_post)
			}
		}
		filelocks["Users/"+friend_name].RUnlock()
	}
	filelocks["Users/" + username].RUnlock()
	fmt.Println("read unlocking Users/", username)
	return user_feed
}

// handler for main feed page, to post and see other's posts
func post_croak(post string, username string) string {
	if len(post) > 100 {
		return "512"
	} else {
		addPostFile(username, post)
		return "200"
	}
}

func getUserFeed(username string) []byte {
	user_feed := createUserFeed(username)
	jsonFeed, _ := json.Marshal(user_feed)
	return jsonFeed
}

// login handler
func login(username string, password string) string {
	// make sure username is associated with an account & sure password is correct
	if !userExists(username) {
		return "601"
	} else if !passwordCorrect(username, password) {
		return "602"
	}
	return "200"
}

// handler for adding and removing friends
func manageFriends(username string, friend string, choice string) string {
	if userExists(friend) {
		if choice == "add" && !areFriends(username, friend) && friend != username {
			addFriendForUser(username, friend)
		} else if choice == "remove" && areFriends(username, friend) && friend != username { // choice is to remove friend
			removeFriendForUser(username, friend)
		}
		return "200"
	} else {
		return "601"
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
		if user != "Users/.DS_Store/friends/"+username+".txt" {
			if _, err := os.Stat(user + "/friends/" + username + ".txt"); os.IsNotExist(err) {
				break
			} else {
				getReadLock("Users/"+user)
				filename := user + "/friends/" + username + ".txt"
				getLock(filename)
				e := os.RemoveAll(filename)
				if e != nil {
					log.Fatal(e)
				}
				filelocks[filename].Unlock()
				filelocks["Users/"+user].RUnlock()
				fmt.Println("unlocked", filename, "in deleteUser")
			}
		}
	}
}

// handler for deleting user account
func deleteAccount(username string) {
	deleteUser(username)
	fmt.Println("delete", username)
}

// handler for making a new account
func makeAccount(username string, name string, email string, password string) string {
	fmt.Println("in server side make accounts")
	if username == "" || name == "" || email == "" || password == "" {
		return "606"
	} else {
		if userExists(username) {
			return "607"
		} else {
			createUserFile(username, name, password, email)
		}
	}

	return "200"
}

func clientConns(listener net.Listener) chan net.Conn {
	clientJobs := make(chan net.Conn)
	go func () {
		for {
			client, err := listener.Accept()
			if err != nil {
				fmt.Println(err)
			}
			clientJobs <- client
		}
	} ()
	return clientJobs
}

func handleConn(client net.Conn) {
	fmt.Println("in handle conn server func")	
	defer client.Close()
	// read message from client
	message := make([]byte, 1024)
	n, err := client.Read(message)
	if err != nil {
		fmt.Println(err)
	}

	// format message into tokens
	s_message := string(message[:n])
	fmt.Println(s_message)
	message_tokens := strings.Split(s_message, ",")
	if message_tokens[0] == "create" {
		fmt.Println("create request")
		statusCode := makeAccount(message_tokens[1], message_tokens[2], message_tokens[3], message_tokens[4])
		fmt.Fprintf(client, statusCode)
		return
	}
	if message_tokens[0] == "delete" {
		fmt.Println("delete request")
		deleteAccount(message_tokens[1])
		return
	}
	if message_tokens[0] == "manage" {
		fmt.Println("manage request")
		statusCode := manageFriends(message_tokens[1], message_tokens[2], message_tokens[3])
		fmt.Fprintf(client, statusCode)
		return
	}
	if message_tokens[0] == "login" {
		fmt.Println("login request")
		statusCode := login(message_tokens[1], message_tokens[2])
		fmt.Fprintf(client, statusCode)
		return
	}
	if message_tokens[0] == "post" {
		fmt.Println("post request")
		statusCode := post_croak(message_tokens[1], message_tokens[2])
		fmt.Fprintf(client, statusCode)
		return
	}
	if message_tokens[0] == "getUserFeed" {
		fmt.Println("get user feed request")
		jsonFeed := getUserFeed(message_tokens[1])
		// fmt.Fprintf(client, "jsonFeed")
		client.Write(jsonFeed)
		return
	}
	
}

func main() {
	// create Users directory where users are stored
	if _, err := os.Stat("Users/"); os.IsNotExist(err) {
		os.Mkdir("Users/", 0777)
	}
	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println(err)
	}
	conns := clientConns(ln)
	for {
		go handleConn(<-conns)
	}


}
