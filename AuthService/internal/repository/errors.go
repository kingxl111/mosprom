package repository

import "errors"

var (
	ErrorBuildSelectQuery = errors.New("failed to build SELECT query")
	ErrorBuildInsertQuery = errors.New("failed to build INSERT query")
	ErrorBuildUpdateQuery = errors.New("failed to build UPDATE query")
	ErrorInsertUser       = errors.New("failed to insert user")
	ErrorInsertToken      = errors.New("failed to insert refresh token")
	ErrorInsertSession    = errors.New("failed to insert session")
	ErrorUpdateUser       = errors.New("failed to update user")
	ErrorUpdateToken      = errors.New("failed to update refresh token")
	ErrorUpdateSession    = errors.New("failed to update session")
	ErrorUserNotFound     = errors.New("user not found")
)
