/*
 * This file is part of the Dicot project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package v3

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dicot-project/dicot-api/pkg/api/identity"
	"github.com/dicot-project/dicot-api/pkg/api/identity/v1"
)

type GroupListRes struct {
	Groups []GroupInfo `json:"groups"`
}

type GroupInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DomainID    string `json:"domain_id"`
}

type GroupCreateReq struct {
	Group GroupInfo `json:"group"`
}

type GroupUpdateReq struct {
	Group GroupUpdateInfo `json:"group"`
}

type GroupUpdateInfo struct {
	ID          string  `json:"id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type GroupShowRes struct {
	Group GroupInfo `json:"group"`
}

func (svc *service) GroupList(c *gin.Context) {
	name := c.Query("name")

	clnt := identity.NewGroupClient(svc.RESTClient, identity.FormatDomainNamespace("default"))

	groups, err := clnt.List()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	res := &GroupListRes{
		Groups: []GroupInfo{},
	}

	// XXX Links field
	for _, group := range groups.Items {
		if name != "" && group.ObjectMeta.Name != name {
			continue
		}
		info := GroupInfo{
			ID:          string(group.ObjectMeta.UID),
			Name:        group.Spec.Name,
			Description: group.Spec.Description,
		}
		res.Groups = append(res.Groups, info)
	}

	c.JSON(http.StatusOK, res)
}

func (svc *service) GroupCreate(c *gin.Context) {
	var req GroupCreateReq
	err := c.BindJSON(&req)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	domClnt := identity.NewProjectClient(svc.RESTClient, v1.NamespaceSystem)

	var domNamespace string
	if req.Group.DomainID != "" {
		dom, err := domClnt.GetByUID(req.Group.DomainID)
		if err != nil {
			if errors.IsNotFound(err) {
				c.AbortWithError(http.StatusBadRequest, err)
			} else {
				c.AbortWithError(http.StatusInternalServerError, err)
			}
			return
		}
		domNamespace = dom.Spec.Namespace
	} else {
		dom, err := domClnt.Get("default")
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		req.Group.DomainID = string(dom.ObjectMeta.UID)
		domNamespace = dom.Spec.Namespace
	}

	clnt := identity.NewGroupClient(svc.RESTClient, domNamespace)

	exists, err := clnt.Exists(req.Group.Name)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if exists {
		c.AbortWithStatus(http.StatusConflict)
		return
	}

	group := &v1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: identity.SanitizeName(req.Group.Name),
		},
		Spec: v1.GroupSpec{
			Name:        req.Group.Name,
			Description: req.Group.Description,
		},
	}

	group, err = clnt.Create(group)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// XXX links
	res := GroupShowRes{
		Group: GroupInfo{
			ID:          string(group.ObjectMeta.UID),
			Name:        group.Spec.Name,
			Description: group.Spec.Description,
		},
	}

	c.JSON(http.StatusCreated, res)
}

func (svc *service) GroupShow(c *gin.Context) {
	id := c.Param("id")

	clnt := identity.NewGroupClient(svc.RESTClient, identity.FormatDomainNamespace("default"))

	group, err := clnt.GetByUID(id)
	if err != nil {
		if errors.IsNotFound(err) {
			c.AbortWithError(http.StatusNotFound, err)
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	// XXX links
	res := GroupShowRes{
		Group: GroupInfo{
			ID:          string(group.ObjectMeta.UID),
			Name:        group.Spec.Name,
			Description: group.Spec.Description,
		},
	}

	c.JSON(http.StatusCreated, res)
}

func (svc *service) GroupUpdate(c *gin.Context) {
	var req GroupUpdateReq
	err := c.BindJSON(&req)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	id := c.Param("id")

	clnt := identity.NewGroupClient(svc.RESTClient, identity.FormatDomainNamespace("default"))

	group, err := clnt.GetByUID(id)
	if err != nil {
		if errors.IsNotFound(err) {
			c.AbortWithError(http.StatusNotFound, err)
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	if req.Group.Name != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if req.Group.Description != nil {
		group.Spec.Description = *req.Group.Description
	}

	group, err = clnt.Update(group)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	res := GroupShowRes{
		Group: GroupInfo{
			ID:          string(group.ObjectMeta.UID),
			Name:        group.Spec.Name,
			Description: group.Spec.Description,
		},
	}

	c.JSON(http.StatusOK, res)
}

func (svc *service) GroupDelete(c *gin.Context) {
	id := c.Param("id")

	clnt := identity.NewGroupClient(svc.RESTClient, identity.FormatDomainNamespace("default"))

	group, err := clnt.GetByUID(id)
	if err != nil {
		if errors.IsNotFound(err) {
			c.AbortWithError(http.StatusNotFound, err)
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	err = clnt.Delete(group.ObjectMeta.Name, nil)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.String(http.StatusNoContent, "")
}