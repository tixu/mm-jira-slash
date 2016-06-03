package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"

	jira "github.com/andygrunwald/go-jira"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var inittoken string

const tmpl = `# JIRA
## Summary
{{.Fields.Summary}}
## Description
{{.Fields.Description}}
## Assignee
{{.Fields.Assignee.DisplayName}}
## Info
|Fields  |Status                   |
|:-------|:------------------------|
|Priority|{{.Fields.Priority.Name}}|
|Status|{{.Fields.Status.Name}}|`

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

var jiraClient jira.Client

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

func testHandler(w http.ResponseWriter, r *http.Request) {
	mm, err := analyseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if inittoken != mm.token {
		log.Printf("inittoken : %s; received token : %s", inittoken, mm.token)
		http.Error(w, "bad token", http.StatusForbidden)
	}

	log.Printf("Received a request %+v", mm)
	issueid := mm.text
	jiraClient, _ := jira.NewClient(nil, "https://jira.smals.be/")

	if res, err := jiraClient.Authentication.AcquireSessionCookie("xz", "H$grs3OmT"); err != nil || res == false {
		http.Error(w, err.Error(), http.StatusInternalServerError)

	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	issue, _, err := jiraClient.Issue.Get(issueid)

	t := template.New("Issue template")
	/*		tmpl := `# JIRA
			## Summary
			{{.Fields.Summary}}
			## Description
			{{.Fields.Description}}
			## Assignee
			{{.Fields.Assignee.DisplayName}}
			## Info
			|Fields  |Status                   |
			|:-------|:------------------------|
			|Priority|{{.Fields.Priority.Name}}|
			|Status|{{.Fields.Status.Name}}|` */
	t, err = t.Parse(tmpl)
	if err != nil {
		panic(err)
	}
	buff := bytes.NewBufferString("")
	t.Execute(buff, issue)
	s, _ := json.Marshal(&Response{Type: "ephemeral", Text: buff.String()})
	log.Println(string(s))
	w.Write(s)

}

func main() {
	tokenPtr := flag.String("token", "", "mattermost token value")
	flag.Parse()
	inittoken = *tokenPtr

	r := mux.NewRouter()
	r.Handle("/", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(testHandler)))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	http.Handle("/", r)
	log.Println("starting server...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
