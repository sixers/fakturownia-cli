package main

import (
	"os"

	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/client"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/selfupdate"
	"github.com/sixers/fakturownia-cli/internal/spec"
)

func main() {
	store, err := auth.NewKeyringStore()
	if err != nil {
		exitWith(err)
	}

	root := spec.NewRootCommand(spec.Dependencies{
		Auth:    auth.NewService(store),
		Client:  client.NewService(store),
		Invoice: invoice.NewService(store),
		Doctor:  doctor.NewService(store),
		Self:    selfupdate.NewService(),
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})

	if err := root.Execute(); err != nil {
		exitWith(err)
	}
}

func exitWith(err error) {
	type exitCoder interface {
		ExitCode() int
	}

	if err == nil {
		return
	}
	if coded, ok := err.(exitCoder); ok {
		os.Exit(coded.ExitCode())
	}
	os.Exit(9)
}
