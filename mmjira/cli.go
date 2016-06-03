package mmjira

import (
	"bytes"
	"html/template"
	"net/url"
	"strings"

	jira "github.com/andygrunwald/go-jira"
)

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

type Cli struct {
	Endpoint   *url.URL
	User       string
	Password   string
	JiraClient *jira.Client
}

func New(endpoint string, user string, password string) (*Cli, error) {
	url, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil {
		return nil, err
	}
	c, _ := jira.NewClient(nil, strings.TrimRight(endpoint, "/"))

	cli := &Cli{
		Endpoint:   url,
		User:       user,
		Password:   password,
		JiraClient: c,
	}
	return cli, nil
}

func (c *Cli) ViewTicket(issueID string) (issueTxt string, err error) {

	if res, err := c.JiraClient.Authentication.AcquireSessionCookie(c.User, c.Password); err != nil || res == false {
		return "", err
	}

	issue, _, err := c.JiraClient.Issue.Get(issueID)
	if err != nil {
		return "", err
	}

	t := template.New("Issue template")

	t, err = t.Parse(tmpl)
	if err != nil {
		panic(err)
	}
	buff := bytes.NewBufferString("")
	t.Execute(buff, issue)
	return buff.String(), nil

}
