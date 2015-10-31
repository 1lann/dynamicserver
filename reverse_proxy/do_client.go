package main

import (
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

var doClient *godo.Client

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func loadDoClient() {
	tokenSource := &TokenSource{
		AccessToken: globalConfig.APIToken,
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	doClient = godo.NewClient(oauthClient)
}
