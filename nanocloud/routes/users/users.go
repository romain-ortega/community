/*
 * Nanocloud Community, a comprehensive platform to turn any application
 * into a cloud solution.
 *
 * Copyright (C) 2015 Nanocloud Software
 *
 * This file is part of Nanocloud community.
 *
 * Nanocloud community is free software; you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * Nanocloud community is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package users

import (
	"encoding/json"
	"errors"

	"github.com/Nanocloud/community/nanocloud/models/ldap"
	"github.com/Nanocloud/community/nanocloud/models/users"
	"github.com/Nanocloud/community/nanocloud/router"
	"github.com/Nanocloud/community/nanocloud/utils"
	log "github.com/Sirupsen/logrus"
)

type hash map[string]interface{}

func Delete(req router.Request) (*router.Response, error) {
	userId := req.Params["id"]
	if len(userId) == 0 {
		return router.JSONResponse(400, hash{
			"error": [1]hash{
				hash{
					"detail": "User id needed for deletion",
				},
			},
		}), nil
	}

	user, err := users.GetUser(userId)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return router.JSONResponse(404, hash{
			"error": [1]hash{
				hash{
					"detail": "User not found",
				},
			},
		}), nil
	}

	if user.IsAdmin {
		return router.JSONResponse(403, hash{
			"error": [1]hash{
				hash{
					"detail": "Admins cannot be deleted",
				},
			},
		}), nil
	}

	err = ldap.DeleteAccount(user.Id)
	if err != nil {
		log.Errorf("Unable to delete user in ad: %s", err.Error())
		switch err {
		case ldap.DeleteFailed:
			return router.JSONResponse(500, hash{
				"error": [1]hash{
					hash{
						"detail": err.Error(),
					},
				},
			}), nil
		case ldap.UnknownUser:
			log.Info("User doesn't exist in AD")
			break
		default:
			return nil, err
		}
	}

	err = users.DeleteUser(user.Id)
	if err != nil {
		log.Errorf("Unable to delete user: ", err.Error())
		return nil, err
	}

	return router.JSONResponse(200, hash{
		"data": hash{
			"success": true,
		},
	}), nil
}

func Disable(req router.Request) (*router.Response, error) {
	userId := req.Params["id"]
	if userId == "" {
		return router.JSONResponse(404, hash{
			"error": [1]hash{
				hash{
					"detail": "User id needed for desactivation",
				},
			},
		}), nil
	}

	exists, err := users.UserExists(userId)
	if err != nil {
		return router.JSONResponse(500, hash{
			"error": [1]hash{
				hash{
					"detail": err.Error(),
				},
			},
		}), nil
	}

	if !exists {
		return router.JSONResponse(404, hash{
			"error": [1]hash{
				hash{
					"detail": "User not found",
				},
			},
		}), nil
	}

	err = users.DisableUser(userId)
	if err != nil {
		log.Errorf("Unable to disable user: %s", err.Error())
		return nil, err
	}

	return router.JSONResponse(200, hash{
		"data": hash{
			"success": true,
		},
	}), nil
}

func Get(req router.Request) (*router.Response, error) {
	users, err := users.FindUsers()
	if err != nil {
		log.Errorf("unable to get user lists: %s", err.Error())
		return nil, err
	}
	return router.JSONResponse(200, hash{"data": users}), nil
}

func Post(req router.Request) (*router.Response, error) {
	var user struct {
		Data struct {
			Email     string
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Password  string
		}
	}

	err := json.Unmarshal([]byte(req.Body), &user)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	newUser, err := users.CreateUser(
		true,
		user.Data.Email,
		user.Data.FirstName,
		user.Data.LastName,
		user.Data.Password,
		false,
	)
	switch err {
	case users.UserDuplicated:
		return router.JSONResponse(409, hash{
			"error": [1]hash{
				hash{
					"detail": err.Error(),
				},
			},
		}), nil
	case users.UserNotCreated:
		return router.JSONResponse(500, hash{
			"error": [1]hash{
				hash{
					"detail": err.Error(),
				},
			},
		}), nil
	}

	winpass := utils.RandomString(8) + "s4D+"
	sam, err := ldap.AddUser(newUser.Id, winpass)
	if err != nil {
		return router.JSONResponse(500, hash{
			"error": [1]hash{
				hash{
					"detail": err.Error(),
				},
			},
		}), nil
	}

	err = users.UpdateUserAd(newUser.Id, sam, winpass)
	if err != nil {
		return nil, err
	}

	return router.JSONResponse(201, hash{
		"data": hash{
			"id": newUser.Id,
		},
	}), nil
}

func Login(req router.Request) (*router.Response, error) {
	var body struct {
		Username string
		Password string
	}

	err := json.Unmarshal(req.Body, &body)
	if err != nil {
		return nil, err
	}

	user, err := users.GetUserFromEmailPassword(body.Username, body.Password)
	switch err {
	case users.InvalidCredentials:
	case users.UserDisabled:
		return router.JSONResponse(400, hash{
			"success": false,
			"error":   err.Error(),
		}), nil

	case nil:
		if user == nil {
			log.Error("unable to log the user in")
			return nil, errors.New("unable to log the user in")
		}

		return router.JSONResponse(200, hash{
			"success": true,
			"user":    user,
		}), nil
	}
	return nil, err
}

func UpdatePassword(req router.Request) (*router.Response, error) {
	userId := req.Params["id"]
	if userId == "" {
		return router.JSONResponse(400, hash{
			"error": "User id needed to modify account",
		}), nil
	}

	var user struct {
		Password string
	}

	err := json.Unmarshal(req.Body, &user)
	if err != nil {
		log.Errorf("Unable to parse body request: %s", err.Error())
		return nil, err
	}

	exists, err := users.UserExists(userId)
	if err != nil {
		log.Errorf("Unable to check user existance: %s", err.Error())
		return nil, err
	}

	if !exists {
		return router.JSONResponse(404, hash{
			"error": "User not found",
		}), nil
	}

	err = users.UpdateUserPassword(userId, user.Password)
	if err != nil {
		log.Errorf("Unable to update user password: %s", err.Error())
		return nil, err
	}

	return router.JSONResponse(200, hash{
		"success": true,
	}), nil
}

func GetUser(req router.Request) (*router.Response, error) {
	userId := req.Params["id"]
	if userId == "" {
		return router.JSONResponse(400, hash{
			"error": "User id needed to retrieve account informations",
		}), nil
	}

	user, err := users.GetUser(userId)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return router.JSONResponse(404, hash{
			"error": "User Not Found",
		}), nil
	}

	return router.JSONResponse(200, user), nil
}
