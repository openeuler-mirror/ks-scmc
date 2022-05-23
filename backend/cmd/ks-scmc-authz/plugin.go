package main

import (
	"log"
	"net/url"
	"strings"

	"github.com/docker/go-plugins-helpers/authorization"
)

type AuthZPlugin struct {
	magicUser string
}

func NewAuthZPlugin(magicUser string) *AuthZPlugin {
	return &AuthZPlugin{magicUser}
}

func (p *AuthZPlugin) containerNeedCheck(id string) bool {
	return globalConfig.isSensitiveContainers(id)
}

func (p *AuthZPlugin) actionNeedCheck(action string) bool {
	switch action {
	case "start", "stop", "restart", "kill", "remove", "update",
		"pause", "unpause", "exec", "rename", "attach", "commit", "export":
		return true
	}
	return false
}

func (p *AuthZPlugin) AuthZReq(req authorization.Request) authorization.Response {
	uri, err := url.QueryUnescape(req.RequestURI)
	if err != nil {
		log.Printf("QueryUnescape uri=%s err=%v", req.RequestURI, err)
		return authorization.Response{Allow: true}
	}

	reqURI, err := url.ParseRequestURI(uri)
	if err != nil {
		log.Printf("ParseRequestURI uri=%s err=%v", uri, err)
		return authorization.Response{Allow: true}
	}

	uriParts := strings.Split(reqURI.String(), "/") // [0]='' [1]=version [2]module(container/image/...)
	// log.Printf("URI=%s, uri parts %v", reqURI.String(), uriParts)

	if (req.RequestMethod == "POST" || req.RequestMethod == "PUT") && len(uriParts) >= 5 {
		if uriParts[2] == "containers" {
			id, action := uriParts[3], uriParts[4]
			if p.actionNeedCheck(action) && p.containerNeedCheck(id) {
				user := req.User
				if strings.Trim(user, " \n\t") == "" {
					if value, ok := req.RequestHeaders["Authz-User"]; ok {
						user = value
					}
				}

				if user == p.magicUser {
					return authorization.Response{Allow: true}
				} else {
					log.Printf("Authz forbid request: %+v", req)
					return authorization.Response{Allow: false, Msg: "Access denied by authz plugin"}
				}
			}
		}
	}

	return authorization.Response{Allow: true}
}

func (plugin *AuthZPlugin) AuthZRes(req authorization.Request) authorization.Response {
	return authorization.Response{Allow: true}
}
