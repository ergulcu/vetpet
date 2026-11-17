// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

// Command vetpet serves the clinic application.
//
//	VETPET_ADDR                      listen address (:8080)
//	VETPET_DATABASE                  SQLite file; without it the data is kept in memory and gone after a restart
//	VETPET_SECURITY_ADMIN_PASSWORD   password of admin (admin)
//	VETPET_SECURITY_USER_PASSWORD    password of user (user)
//	VETPET_SECURE_COOKIE             true when the application is served over HTTPS
//	VETPET_SESSION_TIMEOUT           idle time after which a user is signed out (30m)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ergulcu/vetpet/internal/auth"
	"github.com/ergulcu/vetpet/internal/store"
	"github.com/ergulcu/vetpet/internal/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(ctx, log, os.Getenv); err != nil {
		log.Error("vetpet stopped", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger, getenv func(string) string) error {
	adminPassword := or(getenv("VETPET_SECURITY_ADMIN_PASSWORD"), "admin")
	userPassword := or(getenv("VETPET_SECURITY_USER_PASSWORD"), "user")
	if adminPassword == "admin" || userPassword == "user" {
		log.Warn("demo passwords are in use; set VETPET_SECURITY_ADMIN_PASSWORD and VETPET_SECURITY_USER_PASSWORD")
	}
	secure, err := strconv.ParseBool(or(getenv("VETPET_SECURE_COOKIE"), "false"))
	if err != nil {
		return fmt.Errorf("VETPET_SECURE_COOKIE: %w", err)
	}
	timeout, err := time.ParseDuration(or(getenv("VETPET_SESSION_TIMEOUT"), "30m"))
	if err != nil {
		return fmt.Errorf("VETPET_SESSION_TIMEOUT: %w", err)
	}

	path := getenv("VETPET_DATABASE")
	st, err := store.Open(ctx, path)
	if err != nil {
		return err
	}
	defer st.Close()
	if path == "" {
		log.Info("the data is kept in memory; set VETPET_DATABASE to keep it in a file")
	}

	accounts, err := auth.NewAccounts(adminPassword, userPassword)
	if err != nil {
		return err
	}
	handler, err := web.New(st, accounts, auth.NewSessions(timeout, secure), log)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              or(getenv("VETPET_ADDR"), ":8080"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", srv.Addr)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func or(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
