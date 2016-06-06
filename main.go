package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"git01.smals.be/jira/mmjira"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mattn/go-shellwords"
)

var inittoken string

type cmdError struct {
	cmd  string
	prob string
}

func (e *cmdError) Error() string {
	return fmt.Sprintf("%s - %s", e.cmd, e.prob)
}

// Response is used to store result that must be sent bak to mattermost.
type Response struct {
	Type string `json:"response_type"`
	Text string `json:"text"`
}

//MattermostRequest is a  Request received from mattermost
type MattermostRequest struct {
	channelID   string `schema:"channel_id"`
	channelName string `schema:"channel_name"`
	command     string `schema:"command"`
	responseURL string `schema:"response_url"`
	teamDomain  string `schema:"team_domain"`
	teamID      string `schema:"team_id"`
	text        string `schema:"text"`
	token       string `schema:"token"`
	userID      string `schema:"userID"`
	userName    string `schema:"user_name"`
}

func analyseRequest(r *http.Request) (response MattermostRequest, err error) {
	r.ParseForm()
	response = MattermostRequest{channelID: r.FormValue("channel_id"),
		channelName: r.FormValue("channel_name"),
		command:     r.FormValue("command"),
		responseURL: r.FormValue("response_url"),
		teamDomain:  r.FormValue("team_domain"),
		teamID:      r.FormValue("team_id"),
		text:        r.FormValue("text"),
		token:       r.FormValue("token"),
		userID:      r.FormValue("user_id"),
		userName:    r.FormValue("user_name")}

	return response, err
}

type jiraAction func(args ...string) string

var commands map[string]jiraAction

/*func analysecommand(cmd string) {
	p := shellwords.NewParser()
	args, err := p.Parse(cmd)

	// args should be ["./foo", "bar"]
}*/

func testHandler(w http.ResponseWriter, r *http.Request) {
	mm, err := analyseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if inittoken != mm.token {
		log.Printf("inittoken : %s; received token : %s", inittoken, mm.token)
		http.Error(w, "bad token", http.StatusForbidden)
	}

	args, err := shellwords.Parse(mm.text)

	cmd, args := args[0], args[1:len(args)]
	log.Printf("%s:%#v", cmd, args)

	log.Printf("Received a request %+v", mm)

	c, err := mmjira.New("https://jira.smals.be", "xz", "H$grs3OmT")
	if err != nil {
		http.Error(w, "jira Client creation issue", http.StatusNotFound)

	}
	var issueTxt string

	switch cmd {
	case "VIEW":
		issueTxt, err = c.ViewTicket(args...)
	case "ASSIGN":
		issueTxt, err = c.AssigntoTicket(args...)

	default:
		err = &cmdError{cmd: cmd, prob: "unsupported operation"}
	}

	if err != nil {
		http.Error(w, "jira Client invocation issue", http.StatusNotAcceptable)

	}
	s, _ := json.Marshal(&Response{Type: "ephemeral", Text: issueTxt})
	log.Println(string(s))

	w.Write(s)

}

func main() {
	tokenPtr := flag.String("token", "", "mattermost token value")
	portPtr := flag.Int64("port", 3, "port to listen to")

	flag.Parse()
	inittoken = *tokenPtr
	var s string
	s = ":"
	s += strconv.FormatInt(*portPtr, 10)
	r := mux.NewRouter()
	r.Handle("/jira", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(testHandler)))
	r.PathPrefix("/static").Handler(http.FileServer(http.Dir("./static/")))
	http.Handle("/", r)
	log.Printf("starting server... on %s", s)

	log.Fatal(http.ListenAndServe(s, r))
}
