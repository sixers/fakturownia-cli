package main

import (
	"os"

	"github.com/sixers/fakturownia-cli/internal/account"
	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/category"
	"github.com/sixers/fakturownia-cli/internal/client"
	"github.com/sixers/fakturownia-cli/internal/department"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/issuer"
	"github.com/sixers/fakturownia-cli/internal/payment"
	"github.com/sixers/fakturownia-cli/internal/pricelist"
	"github.com/sixers/fakturownia-cli/internal/product"
	"github.com/sixers/fakturownia-cli/internal/recurring"
	"github.com/sixers/fakturownia-cli/internal/selfupdate"
	"github.com/sixers/fakturownia-cli/internal/spec"
	"github.com/sixers/fakturownia-cli/internal/user"
	"github.com/sixers/fakturownia-cli/internal/warehouse"
	"github.com/sixers/fakturownia-cli/internal/warehouseaction"
	"github.com/sixers/fakturownia-cli/internal/warehousedocument"
	"github.com/sixers/fakturownia-cli/internal/webhook"
)

func main() {
	store, err := auth.NewKeyringStore()
	if err != nil {
		exitWith(err)
	}

	root := spec.NewRootCommand(spec.Dependencies{
		Account:         account.NewService(store),
		Auth:            auth.NewService(store),
		Department:      department.NewService(store),
		Category:        category.NewService(store),
		Client:          client.NewService(store),
		Invoice:         invoice.NewService(store),
		Issuer:          issuer.NewService(store),
		Payment:         payment.NewService(store),
		PriceList:       pricelist.NewService(store),
		Product:         product.NewService(store),
		Recurring:       recurring.NewService(store),
		User:            user.NewService(store),
		Webhook:         webhook.NewService(store),
		Warehouses:      warehouse.NewService(store),
		WarehouseAction: warehouseaction.NewService(store),
		Warehouse:       warehousedocument.NewService(store),
		Doctor:          doctor.NewService(store),
		Self:            selfupdate.NewService(),
		Stdout:          os.Stdout,
		Stderr:          os.Stderr,
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
