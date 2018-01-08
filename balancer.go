// Author: Lisa Frankel
// Load Balancer


package main

import (
	"fmt"
	"net"
	"strings"
	"math"
	"time"
	"path/filepath"
	"io/ioutil"
	"os"
)


// struct that holds a server object - refers to the backend servers
type Server struct {
	port string // 8081, 8082, 8083
	status bool // true = running, false = down
	clients float64 // number of clients connected to servers
	num int // server num
	down time.Time
	up time.Time
}

// initialize all the server objects (there are 3 backend servers)
var S1 Server 
var S2 Server 
var S3 Server


// check errors
func check(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// accept client connections and add them to client jobs
func clientConns(listener net.Listener) chan net.Conn {
	clientJobs := make(chan net.Conn)
	go func () {
		for {
			client, err := listener.Accept()
			check(err)
			clientJobs <- client
		}
	} ()
	return clientJobs
}

// returns a server for the client to read from
// sends the client a server with the least clients
// if all servers are down, it returns "nil"
func getServer() string {
	if !S1.status && !S2.status && !S3.status {
		return "nil"
	}

	if S1.clients == S2.clients && S1.clients == S3.clients {
		return S1.port
	} 
	
	min := math.Min(math.Min(S1.clients, S2.clients), S3.clients)
	
	if S1.clients == min {
		return S1.port
	} else if S2.clients == min {
		return S2.port
	} else if S3.clients == min {
		return S3.port
	}

	return "nil"
}

// incrememnts the number of clients connected to a backend server
func incrmServer(server_num string) {
	if server_num == S1.port {
		S1.clients++
	} else if server_num == S2.port {
		S2.clients++
	} else if server_num == S3.port {
		S3.clients++
	}
}


// sends the client a list of ports to connect to for a write 
// the ports are those of ALL the running servers
// if a server is down, "nil" is appended to the return string
func getWriteServers() string {
	var servers string
	if S1.status {
		servers += "8081,"
		incrmServer(S1.port)
	} else {
		servers += "nil,"
	}
	if S2.status {
		servers += "8082,"
		incrmServer(S2.port)
	} else {
		servers += "nil,"
	}
	if S3.status {
		servers += "8083"
		incrmServer(S3.port)
	} else {
		servers += "nil"
	}

	fmt.Println(servers)
	return servers
}

// changes a servers status to be false (i.e. not running)
// records what time the server went down
func changeStatusToDown(s Server) {
	if s.num == S1.num {
		S1.status = false
		S1.down = time.Now()
		return
	}
	if s.num == S2.num {
		S2.status = false
		S2.down = time.Now()
		return
	}
	if s.num == S3.num {
		S3.status = false
		S3.down = time.Now()
		return
	}
}

// copys the file data from copied_filepath to updating_file
func copyFile(updating_file string, copied_filepath string) {
	file_data, err := ioutil.ReadFile(copied_filepath)

	check(err)
	fmt.Println("copying data: ", string(file_data))
	updateFile, err1 := os.OpenFile(updating_file, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	check(err1)
	_, err2 := updateFile.Write(file_data)
	check(err2)
	updateFile.Close()


}

// restores a database from a dropped server by copying the database of the server which was the last to go down
func copyData(updatingServer string, copiedServer string) {
	fmt.Println("copying data")
	updatingUsersFile := "Users" + updatingServer + "/"
	fmt.Println("dir to be updated:", updatingUsersFile)
	os.RemoveAll(updatingUsersFile)
	os.Mkdir(updatingUsersFile, 0777)
	users, _ := filepath.Glob("Users" + copiedServer + "/*")
	for _, user := range users {
		user_name := user[len("Users" + copiedServer + "/") : len(user)]
		skip := false
		// have to check for this because of the MAC os
		if user_name == ".DS_Store" {
			skip = true
		}
		if !skip {
			user_file := updatingUsersFile + user_name
			os.Mkdir(user_file, 0777) // create that user file
			os.Mkdir(user_file + "/friends", 0777) // create their friends directory
			os.Mkdir(user_file + "/posts", 0777) // create their posts directory
			p1, _ := os.Create(user_file + "/password.txt")
			n1, _ := os.Create(user_file + "/name.txt")
			e1, _ := os.Create(user_file + "/email.txt")
			p1.Close()
			n1.Close()
			e1.Close()
			copyFile(user_file + "/password.txt", user + "/password.txt")
			copyFile(user_file + "/name.txt", user + "/name.txt")
			copyFile(user_file + "/email.txt", user + "/email.txt")
			user_friends, _ := filepath.Glob(user + "/friends/*")
			for _, friend := range user_friends {
				friend_name := friend[len("UsersN/" + user_name + "/friends/") : len(friend) - 4]
				new_friend_file := user_file + "/friends/" + friend_name + ".txt"
				f, _ := os.Create(new_friend_file)
				f.Close()
			}
			user_posts, _ := filepath.Glob(user + "/posts/*")
			for _, post := range user_posts {
				post_name := post[len("UsersN/" + user_name + "/posts/") : len(post) - 4]
				new_post_file := user_file + "/posts/" + post_name + ".txt"
				f, _ := os.Create(new_post_file)
				f.Close()
				copyFile(new_post_file, post)
			}
		}
	}
}

// checks if all three servers are back up and running
// does not stop checking until all servers are up
func allServersRunning () bool {
	for {
		_, err1 := net.Dial("tcp", "localhost:8081")
		_, err2 := net.Dial("tcp", "localhost:8082")
		_, err3 := net.Dial("tcp", "localhost:8083")
		if (err1 == nil && err2 == nil && err3 == nil) {
			fmt.Println("all servers are running again")
			S1.status = true
			S2.status = true
			S3.status = true
			return true
		}

		time.Sleep(1000 * time.Millisecond)
	}
}


// returns the latest down time.Time value of S1, S2, S3
func latestTime() time.Time {
	if S1.down.After(S2.down) && S1.down.After(S3.down) {
		return S1.down
	} else if S2.down.After(S1.down) && S2.down.After(S3.down) {
		return S2.down
	} else {
		return S3.down
	}
}

// find last server to go down // assumes all servers are down
func mostCurrentServer() string {
	mostCurrentServer := latestTime()
	if S1.down == mostCurrentServer {
		return "1"
	} else if S2.down == mostCurrentServer {
		return "2"
	} else {
		return "3"
	}

}

// this calls copyData which restores the servers that crashed databases with the most recently live database
func updateData() {
	fmt.Println("in updating data")

	// to update data, all the servers need to be down
	// this ensures no updates to a DB happen while data is being copied
	// and ensures that two processes aren't trying to access the databases at the same time
	if !S1.status && !S2.status && !S3.status {
		
		// find last server to go down (i.e. most up to date database)
		MCS := mostCurrentServer()

		// copy the data from mostCurrentServers DB to the other servers databases
		if MCS == "1" {
			copyData("2", "1")
			copyData("3", "1")
		} else if MCS == "2" {
			copyData("1", "2")
			copyData("3", "2")
		} else if MCS == "3" {
			copyData("1", "3")
			copyData("2", "3")
		}

		// check if all servers are up and running
		// if/when they are all running, their status is set to true..
		// and the balancer will let clients connect to the servers again
		if allServersRunning() {
			return
		}
	} else { // can't do any data updates if there are still servers running that clients are connecting to
		return
	}
}


// check to see if servers that went down have come back up
func checkDownServers() {
	for {
		fmt.Println("checking down servers")
		
		// see if a server goes from being down to back up
		S1_was_down_is_up := false
		S2_was_down_is_up := false
		S3_was_down_is_up := false

		if !S1.status {
			_, err := net.Dial("tcp", "localhost:8081")
			if err == nil {
				fmt.Println("S1 back up")
				S1_was_down_is_up = true
				S1.up = time.Now()
			}
		}
		if !S2.status {
			_, err := net.Dial("tcp", "localhost:8082")
			if err == nil {
				fmt.Println("S2 back up")
				S2_was_down_is_up = true
				S2.up = time.Now()
			}
		}
		if !S3.status {
			_, err := net.Dial("tcp", "localhost:8083")
			if err == nil {
				fmt.Println("S3 back up")
				S3_was_down_is_up = true
				S3.up = time.Now()
			}

		}


		// if any of the down servers were down and have come back, see if database can be restored
		if (S1_was_down_is_up) || (S2_was_down_is_up) || (S3_was_down_is_up) {
			updateData()
		}


		time.Sleep(1000 * time.Millisecond)
	}
}


// check to see if running servers have crashed/gone down
func checkUpServers() {
	for {
		fmt.Println("checking up servers")
		if S1.status {
			_, err1 := net.Dial("tcp", "localhost:8081")
			if err1 != nil {
				fmt.Println("server 1 down")
				changeStatusToDown(S1)
			}
		}
		if S2.status {
			_, err2 := net.Dial("tcp", "localhost:8082")
			if err2 != nil {
				fmt.Println("server 2 down")
				changeStatusToDown(S2)
			}
		}
		if S3.status {
			_, err3 := net.Dial("tcp", "localhost:8083")
			if err3 != nil {
				fmt.Println("server 3 down")
				changeStatusToDown(S3)
			}
		}

		time.Sleep(1000 * time.Millisecond)
	}
}

// handle incoming connections
func handleConn(client net.Conn) {
	fmt.Println("in handle conn balancer func")	
	defer client.Close()

	// read message from client
	message := make([]byte, 1024)
	n, err := client.Read(message)
	check(err)

	// format message into tokens
	s_message := string(message[:n])
	fmt.Println(s_message)
	message_tokens := strings.Split(s_message, ",")

	if message_tokens[0] == "read" {
		server_num := getServer()
		fmt.Fprintf(client, server_num)
		fmt.Println(server_num)
		incrmServer(server_num)
		return
	}
	if message_tokens[0] == "write" {
		fmt.Println("write servers request")
		servers := getWriteServers()
		fmt.Fprintf(client, servers)
		return
	}

	if message_tokens[0] == "down" {
		fmt.Println("server down", message_tokens[1])
		if message_tokens[1] == "8081"{
			changeStatusToDown(S1)
		} else if message_tokens[1] == "8082" {
			changeStatusToDown(S2)
		} else if message_tokens[1] == "8083" {
			changeStatusToDown(S3)
		}
		return
	}

	if message_tokens[0] == "server1" {
		S1.clients--
		return
	}
	if message_tokens[0] == "server2" {
		S2.clients--	
		return
	}
	if message_tokens[0] == "server3" {
		S3.clients--		
		return
	}

	return
	
}

// initalize the servers information, and listen for connections, and handle them
func main() {
	// define servers
	S1.port = "8081"; S1.status = true; S1.num = 1
	S2.port = "8082"; S2.status = true; S2.num = 2
	S3.port = "8083"; S3.status = true; S3.num = 3

	// create Users directory where users are stored
	ln, err := net.Listen("tcp", ":8084")
	check(err)
	conns := clientConns(ln)
	go checkDownServers()
	go checkUpServers()
	for {
		go handleConn(<-conns)
	}


}
