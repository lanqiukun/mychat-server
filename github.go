package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type GitHubToken struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorUri         string `json:"error_uri"`

	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type GitHubUserInfo struct {
	Login             string `json:"login"`
	Id                uint64 `json:"id"`
	NodeId            string `json:"node_id"`
	AvatarUrl         string `json:"avatar_url"`
	GravatarId        string `json:"gravatar_id"`
	Url               string `json:"url"`
	HtmlUrl           string `json:"html_url"`
	FollowersUrl      string `json:"followers_url"`
	SubscriptionsUrl  string `json:"subscriptions_url"`
	OrganizationsUrl  string `json:"organizations_url"`
	ReposUrl          string `json:"repos_url"`
	ReceivedEventsUrl string `json:"received_events_url"`
	Type              string `json:"Type"`
	Blog              string `json:"blog"`
	PublicRepos       uint64 `json:"public_repos"`
	PublicGists       uint64 `json:"public_gists"`
	Followers         uint64 `json:"followers"`
	Following         uint64 `json:"following"`
}

func requestGitHubToken(code string) (GitHubToken, error) {
	client := &http.Client{}

	url := "https://github.com/login/oauth/access_token?client_id=ba3fb2199f63b790df64&client_secret=c24af06e523f3452eb390da3d5d0b74480aa1761&code=" + code

	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return GitHubToken{}, fmt.Errorf("服务器创建GitHub请求时发生错误")
	}
	request.Header.Add("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return GitHubToken{}, fmt.Errorf("服务器发起GitHub请求时发生错误")
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return GitHubToken{}, fmt.Errorf("服务器读取GitHub响应时发生错误")
	}

	var ght GitHubToken

	err = json.Unmarshal(body, &ght)
	if err != nil {
		return GitHubToken{}, fmt.Errorf("服务器解析GitHub响应时发生错误")
	}

	if ght.Error == "" && ght.ErrorDescription == "" && ght.ErrorUri == "" {
		return ght, nil
	} else {
		return GitHubToken{}, fmt.Errorf("服务器接收到来自客户端的无效令牌: %s", ght.ErrorDescription)
	}

}

func getGitHubUserInfo(ght GitHubToken) (GitHubUserInfo, error) {
	client := &http.Client{}

	url := "https://api.github.com/user"

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return GitHubUserInfo{}, fmt.Errorf("服务器创建GitHub请求时发生错误")
	}

	request.Header.Add("Accept", "application/json")
	request.Header.Add("Authorization", ght.TokenType+" "+ght.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return GitHubUserInfo{}, fmt.Errorf("服务器向GitHub请求用户信息时发生错误")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return GitHubUserInfo{}, fmt.Errorf("服务器读取GitHub响应的用户信息时发生错误")
	}
	defer response.Body.Close()

	var ghui GitHubUserInfo

	err = json.Unmarshal(body, &ghui)
	if err != nil {
		return GitHubUserInfo{}, fmt.Errorf("服务器解析GitHub响应的用户信息时发生错误")
	}

	return ghui, nil
}
