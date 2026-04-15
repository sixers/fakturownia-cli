package spec

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/client"
	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/jsoninput"
	"github.com/sixers/fakturownia-cli/internal/output"
	"github.com/sixers/fakturownia-cli/internal/pricelist"
	"github.com/sixers/fakturownia-cli/internal/product"
	"github.com/sixers/fakturownia-cli/internal/recurring"
	"github.com/sixers/fakturownia-cli/internal/selfupdate"
	"github.com/sixers/fakturownia-cli/internal/warehousedocument"
)

type AuthService interface {
	Login(context.Context, auth.LoginRequest) (*auth.LoginResult, error)
	Status(context.Context, auth.StatusRequest) (*auth.StatusResult, error)
	Logout(context.Context, auth.LogoutRequest) (*auth.LogoutResult, error)
}

type InvoiceService interface {
	List(context.Context, invoice.ListRequest) (*invoice.ListResponse, error)
	Get(context.Context, invoice.GetRequest) (*invoice.GetResponse, error)
	Download(context.Context, invoice.DownloadRequest) (*invoice.DownloadResponse, error)
	Create(context.Context, invoice.CreateRequest) (*invoice.CreateResponse, error)
	Update(context.Context, invoice.UpdateRequest) (*invoice.UpdateResponse, error)
	Delete(context.Context, invoice.DeleteRequest) (*invoice.DeleteResponse, error)
	SendEmail(context.Context, invoice.SendEmailRequest) (*invoice.SendEmailResponse, error)
	ChangeStatus(context.Context, invoice.ChangeStatusRequest) (*invoice.ChangeStatusResponse, error)
	Cancel(context.Context, invoice.CancelRequest) (*invoice.CancelResponse, error)
	PublicLink(context.Context, invoice.PublicLinkRequest) (*invoice.PublicLinkResponse, error)
	AddAttachment(context.Context, invoice.AddAttachmentRequest) (*invoice.AddAttachmentResponse, error)
	DownloadAttachments(context.Context, invoice.DownloadAttachmentsRequest) (*invoice.DownloadAttachmentsResponse, error)
	FiscalPrint(context.Context, invoice.FiscalPrintRequest) (*invoice.FiscalPrintResponse, error)
}

type ClientService interface {
	List(context.Context, client.ListRequest) (*client.ListResponse, error)
	Get(context.Context, client.GetRequest) (*client.GetResponse, error)
	Create(context.Context, client.CreateRequest) (*client.CreateResponse, error)
	Update(context.Context, client.UpdateRequest) (*client.UpdateResponse, error)
	Delete(context.Context, client.DeleteRequest) (*client.DeleteResponse, error)
}

type ProductService interface {
	List(context.Context, product.ListRequest) (*product.ListResponse, error)
	Get(context.Context, product.GetRequest) (*product.GetResponse, error)
	Create(context.Context, product.CreateRequest) (*product.CreateResponse, error)
	Update(context.Context, product.UpdateRequest) (*product.UpdateResponse, error)
}

type PriceListService interface {
	List(context.Context, pricelist.ListRequest) (*pricelist.ListResponse, error)
	Get(context.Context, pricelist.GetRequest) (*pricelist.GetResponse, error)
	Create(context.Context, pricelist.CreateRequest) (*pricelist.CreateResponse, error)
	Update(context.Context, pricelist.UpdateRequest) (*pricelist.UpdateResponse, error)
	Delete(context.Context, pricelist.DeleteRequest) (*pricelist.DeleteResponse, error)
}

type RecurringService interface {
	List(context.Context, recurring.ListRequest) (*recurring.ListResponse, error)
	Create(context.Context, recurring.CreateRequest) (*recurring.CreateResponse, error)
	Update(context.Context, recurring.UpdateRequest) (*recurring.UpdateResponse, error)
}

type WarehouseDocumentService interface {
	List(context.Context, warehousedocument.ListRequest) (*warehousedocument.ListResponse, error)
	Get(context.Context, warehousedocument.GetRequest) (*warehousedocument.GetResponse, error)
	Create(context.Context, warehousedocument.CreateRequest) (*warehousedocument.CreateResponse, error)
	Update(context.Context, warehousedocument.UpdateRequest) (*warehousedocument.UpdateResponse, error)
	Delete(context.Context, warehousedocument.DeleteRequest) (*warehousedocument.DeleteResponse, error)
}

type DoctorService interface {
	Run(context.Context, doctor.RunRequest) (*doctor.RunResult, error)
}

type SelfUpdateService interface {
	Update(context.Context, selfupdate.UpdateRequest) (*selfupdate.UpdateResult, error)
}

type Dependencies struct {
	Auth      AuthService
	Client    ClientService
	Invoice   InvoiceService
	Product   ProductService
	PriceList PriceListService
	Recurring RecurringService
	Warehouse WarehouseDocumentService
	Doctor    DoctorService
	Self      SelfUpdateService
	Stdout    io.Writer
	Stderr    io.Writer
}

type globalOptions struct {
	Profile        string
	JSON           bool
	Output         string
	Quiet          bool
	Fields         []string
	Columns        []string
	Raw            bool
	DryRun         bool
	TimeoutMS      int
	MaxRetries     int
	NonInteractive bool
	Config         string
}

func NewRootCommand(deps Dependencies) *cobra.Command {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}

	globals := globalOptions{}
	root := &cobra.Command{
		Use:           "fakturownia",
		Short:         "Agent-first CLI for the Fakturownia API",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(deps.Stdout)
	root.SetErr(deps.Stderr)
	root.PersistentFlags().StringVar(&globals.Profile, "profile", "", "select a named profile")
	root.PersistentFlags().BoolVar(&globals.JSON, "json", false, "alias for --output json")
	root.PersistentFlags().StringVar(&globals.Output, "output", "human", "output format: human|json")
	root.PersistentFlags().BoolVarP(&globals.Quiet, "quiet", "q", false, "emit bare values when exactly one field or column remains")
	root.PersistentFlags().StringSliceVar(&globals.Fields, "fields", nil, "project JSON envelope data fields using dot/bracket paths like number or positions[].name")
	root.PersistentFlags().StringSliceVar(&globals.Columns, "columns", nil, "select human table columns using dot/bracket paths like number or positions[].name")
	root.PersistentFlags().BoolVar(&globals.Raw, "raw", false, "emit the upstream JSON response body directly when supported")
	root.PersistentFlags().BoolVar(&globals.DryRun, "dry-run", false, "accepted on read-only commands and reserved for future mutating previews")
	root.PersistentFlags().IntVar(&globals.TimeoutMS, "timeout-ms", 30000, "HTTP timeout in milliseconds")
	root.PersistentFlags().IntVar(&globals.MaxRetries, "max-retries", 2, "maximum retries for idempotent reads")
	root.PersistentFlags().BoolVar(&globals.NonInteractive, "non-interactive", true, "disable interactive behavior")
	root.PersistentFlags().StringVar(&globals.Config, "config", "", "override the config file path")

	root.AddCommand(newAuthCommand(deps, &globals))
	root.AddCommand(newClientCommand(deps, &globals))
	root.AddCommand(newInvoiceCommand(deps, &globals))
	root.AddCommand(newProductCommand(deps, &globals))
	root.AddCommand(newPriceListCommand(deps, &globals))
	root.AddCommand(newRecurringCommand(deps, &globals))
	root.AddCommand(newWarehouseDocumentCommand(deps, &globals))
	root.AddCommand(newDoctorCommand(deps, &globals))
	root.AddCommand(newSelfCommand(deps, &globals))
	root.AddCommand(newSchemaCommand(deps, &globals))
	root.Version = Version
	root.SetVersionTemplate("{{printf \"%s\\n\" .Version}}")
	return root
}

func newAuthCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Persist and inspect credentials",
	}

	loginSpec, _ := FindCommand("auth", "login")
	var loginReq auth.LoginRequest
	loginCmd := &cobra.Command{
		Use:   loginSpec.Use,
		Short: loginSpec.Short,
		Long:  BuildLongDescription(loginSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, loginSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "auth login"}, appErr)
			}
			if loginReq.APIToken == "" {
				loginReq.APIToken = config.LookupEnv().APIToken
			}
			if loginReq.URL == "" && loginReq.Prefix == "" {
				loginReq.URL = config.LookupEnv().URL
			}
			loginReq.ConfigPath = globals.Config
			loginReq.Profile = globals.Profile

			start := time.Now()
			result, err := deps.Auth.Login(cmd.Context(), loginReq)
			meta := output.Meta{
				Command:    "auth login",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:     result,
				Warnings: warnings,
				Meta:     meta,
				HumanRenderer: output.LinesRenderer{
					Lines: func(data any) ([]string, error) {
						res := data.(*auth.LoginResult)
						return []string{
							fmt.Sprintf("profile: %s", res.Profile),
							fmt.Sprintf("url: %s", res.URL),
							fmt.Sprintf("default_profile: %s", res.DefaultProfile),
						}, nil
					},
				},
			})
		},
	}
	loginCmd.Flags().StringVar(&loginReq.URL, "url", "", "explicit HTTPS account URL")
	loginCmd.Flags().StringVar(&loginReq.Prefix, "prefix", "", "account prefix")
	loginCmd.Flags().StringVar(&loginReq.APIToken, "api-token", "", "Fakturownia API token")
	loginCmd.Flags().BoolVar(&loginReq.SetDefault, "set-default", false, "mark the saved profile as default")

	statusSpec, _ := FindCommand("auth", "status")
	statusCmd := &cobra.Command{
		Use:   statusSpec.Use,
		Short: statusSpec.Short,
		Long:  BuildLongDescription(statusSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, statusSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "auth status"}, appErr)
			}
			start := time.Now()
			result, err := deps.Auth.Status(cmd.Context(), auth.StatusRequest{
				ConfigPath: globals.Config,
				Profile:    globals.Profile,
				Env:        config.LookupEnv(),
			})
			meta := output.Meta{
				Command:    "auth status",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          result,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}

	logoutSpec, _ := FindCommand("auth", "logout")
	var yes bool
	logoutCmd := &cobra.Command{
		Use:   logoutSpec.Use,
		Short: logoutSpec.Short,
		Long:  BuildLongDescription(logoutSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, logoutSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "auth logout"}, appErr)
			}
			if !yes {
				return writeCommandError(cmd, opts, output.Meta{Command: "auth logout"}, output.Usage("confirmation_required", "--yes is required for auth logout", "rerun with --yes to remove the stored profile"))
			}
			start := time.Now()
			result, err := deps.Auth.Logout(cmd.Context(), auth.LogoutRequest{
				ConfigPath: globals.Config,
				Profile:    globals.Profile,
				Env:        config.LookupEnv(),
			})
			meta := output.Meta{
				Command:    "auth logout",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:     result,
				Warnings: warnings,
				Meta:     meta,
				HumanRenderer: output.LinesRenderer{
					Lines: func(data any) ([]string, error) {
						res := data.(*auth.LogoutResult)
						return []string{fmt.Sprintf("removed profile %s", res.Profile)}, nil
					},
				},
			})
		},
	}
	logoutCmd.Flags().BoolVar(&yes, "yes", false, "confirm profile removal")

	authCmd.AddCommand(loginCmd, statusCmd, logoutCmd)
	return authCmd
}

func newClientCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "Read and manage clients",
	}

	listSpec, _ := FindCommand("client", "list")
	listReq := client.ListRequest{Page: 1, PerPage: 25}
	listCmd := &cobra.Command{
		Use:   listSpec.Use,
		Short: listSpec.Short,
		Long:  BuildLongDescription(listSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, listSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "client list"}, appErr)
			}
			listReq.ConfigPath = globals.Config
			listReq.Profile = globals.Profile
			listReq.Env = config.LookupEnv()
			listReq.Timeout = timeoutFromGlobals(globals)
			listReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Client.List(cmd.Context(), listReq)
			meta := output.Meta{
				Command:    "client list",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
				meta.Pagination = &result.Pagination
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:           result.Clients,
				RawBody:        result.RawBody,
				Warnings:       warnings,
				Meta:           meta,
				HumanRenderer:  output.TableRenderer{},
				DefaultColumns: defaultColumns(listSpec, []string{"id", "name", "tax_no", "email", "city", "country"}),
			})
		},
	}
	listCmd.Flags().IntVar(&listReq.Page, "page", 1, "requested result page")
	listCmd.Flags().IntVar(&listReq.PerPage, "per-page", 25, "requested result count per page")
	listCmd.Flags().StringVar(&listReq.Name, "name", "", "filter by client name")
	listCmd.Flags().StringVar(&listReq.Email, "email", "", "filter by client email")
	listCmd.Flags().StringVar(&listReq.Shortcut, "shortcut", "", "filter by client shortcut")
	listCmd.Flags().StringVar(&listReq.TaxNo, "tax-no", "", "filter by client tax number")
	listCmd.Flags().StringVar(&listReq.ExternalID, "external-id", "", "filter by external client ID")

	getSpec, _ := FindCommand("client", "get")
	var getReq client.GetRequest
	getCmd := &cobra.Command{
		Use:   getSpec.Use,
		Short: getSpec.Short,
		Long:  BuildLongDescription(getSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, getSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "client get"}, appErr)
			}
			getReq.ConfigPath = globals.Config
			getReq.Profile = globals.Profile
			getReq.Env = config.LookupEnv()
			getReq.Timeout = timeoutFromGlobals(globals)
			getReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Client.Get(cmd.Context(), getReq)
			meta := output.Meta{
				Command:    "client get",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          result.Client,
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	getCmd.Flags().StringVar(&getReq.ID, "id", "", "client ID")
	getCmd.Flags().StringVar(&getReq.ExternalID, "external-id", "", "external client ID")

	createSpec, _ := FindCommand("client", "create")
	var createInput string
	createCmd := &cobra.Command{
		Use:   createSpec.Use,
		Short: createSpec.Short,
		Long:  BuildLongDescription(createSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, createSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "client create"}, appErr)
			}
			input, err := client.ParseInput(createInput, cmd.InOrStdin())
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "client create"}, err)
			}

			start := time.Now()
			result, err := deps.Client.Create(cmd.Context(), client.CreateRequest{
				ConfigPath: globals.Config,
				Profile:    globals.Profile,
				Env:        config.LookupEnv(),
				Timeout:    timeoutFromGlobals(globals),
				MaxRetries: globals.MaxRetries,
				Input:      input,
				DryRun:     globals.DryRun,
			})
			meta := output.Meta{
				Command:    "client create",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          clientCreateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	createCmd.Flags().StringVar(&createInput, "input", "", "client JSON input as inline JSON, @file, or - for stdin")
	_ = createCmd.MarkFlagRequired("input")

	updateSpec, _ := FindCommand("client", "update")
	var updateReq client.UpdateRequest
	var updateInput string
	updateCmd := &cobra.Command{
		Use:   updateSpec.Use,
		Short: updateSpec.Short,
		Long:  BuildLongDescription(updateSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, updateSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "client update"}, appErr)
			}
			input, err := client.ParseInput(updateInput, cmd.InOrStdin())
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "client update"}, err)
			}

			updateReq.ConfigPath = globals.Config
			updateReq.Profile = globals.Profile
			updateReq.Env = config.LookupEnv()
			updateReq.Timeout = timeoutFromGlobals(globals)
			updateReq.MaxRetries = globals.MaxRetries
			updateReq.Input = input
			updateReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Client.Update(cmd.Context(), updateReq)
			meta := output.Meta{
				Command:    "client update",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          clientUpdateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	updateCmd.Flags().StringVar(&updateReq.ID, "id", "", "client ID")
	updateCmd.Flags().StringVar(&updateInput, "input", "", "client JSON input as inline JSON, @file, or - for stdin")
	_ = updateCmd.MarkFlagRequired("id")
	_ = updateCmd.MarkFlagRequired("input")

	deleteSpec, _ := FindCommand("client", "delete")
	var deleteReq client.DeleteRequest
	var deleteYes bool
	deleteCmd := &cobra.Command{
		Use:   deleteSpec.Use,
		Short: deleteSpec.Short,
		Long:  BuildLongDescription(deleteSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, deleteSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "client delete"}, appErr)
			}
			if !deleteYes {
				return writeCommandError(cmd, opts, output.Meta{Command: "client delete"}, output.Usage("confirmation_required", "--yes is required for client delete", "rerun with --yes to delete the client"))
			}

			deleteReq.ConfigPath = globals.Config
			deleteReq.Profile = globals.Profile
			deleteReq.Env = config.LookupEnv()
			deleteReq.Timeout = timeoutFromGlobals(globals)
			deleteReq.MaxRetries = globals.MaxRetries
			deleteReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Client.Delete(cmd.Context(), deleteReq)
			meta := output.Meta{
				Command:    "client delete",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)

			humanRenderer := output.HumanRenderer(output.LinesRenderer{
				Lines: func(data any) ([]string, error) {
					res := data.(*client.DeleteResponse)
					return []string{fmt.Sprintf("deleted client %s", res.ID)}, nil
				},
			})
			data := clientDeleteData(result)
			if result.DryRun != nil {
				humanRenderer = output.JSONRenderer{}
			}
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          data,
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: humanRenderer,
			})
		},
	}
	deleteCmd.Flags().StringVar(&deleteReq.ID, "id", "", "client ID")
	deleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "confirm client deletion")
	_ = deleteCmd.MarkFlagRequired("id")

	clientCmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, deleteCmd)
	return clientCmd
}

func newInvoiceCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	invoiceCmd := &cobra.Command{
		Use:   "invoice",
		Short: "Read and manage invoices",
	}

	listSpec, _ := FindCommand("invoice", "list")
	listReq := invoice.ListRequest{Page: 1, PerPage: 25}
	listCmd := &cobra.Command{
		Use:   listSpec.Use,
		Short: listSpec.Short,
		Long:  BuildLongDescription(listSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, listSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice list"}, appErr)
			}
			listReq.ConfigPath = globals.Config
			listReq.Profile = globals.Profile
			listReq.Env = config.LookupEnv()
			listReq.Timeout = timeoutFromGlobals(globals)
			listReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Invoice.List(cmd.Context(), listReq)
			meta := output.Meta{
				Command:    "invoice list",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
				meta.Pagination = &result.Pagination
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:           result.Invoices,
				RawBody:        result.RawBody,
				Warnings:       warnings,
				Meta:           meta,
				HumanRenderer:  output.TableRenderer{},
				DefaultColumns: defaultColumns(listSpec, []string{"id", "number", "issue_date", "buyer_name", "price_gross", "status"}),
			})
		},
	}
	listCmd.Flags().IntVar(&listReq.Page, "page", 1, "requested result page")
	listCmd.Flags().IntVar(&listReq.PerPage, "per-page", 25, "requested result count per page")
	listCmd.Flags().StringVar(&listReq.Period, "period", "", "date period filter")
	listCmd.Flags().StringVar(&listReq.DateFrom, "date-from", "", "lower date bound for period=more")
	listCmd.Flags().StringVar(&listReq.DateTo, "date-to", "", "upper date bound for period=more")
	listCmd.Flags().BoolVar(&listReq.IncludePositions, "include-positions", false, "include invoice positions")
	listCmd.Flags().StringVar(&listReq.ClientID, "client-id", "", "filter by client ID")
	listCmd.Flags().StringSliceVar(&listReq.InvoiceIDs, "invoice-ids", nil, "filter by specific invoice IDs")
	listCmd.Flags().StringVar(&listReq.Number, "number", "", "filter by invoice number")
	listCmd.Flags().StringSliceVar(&listReq.Kinds, "kind", nil, "filter by invoice kind")
	listCmd.Flags().StringVar(&listReq.SearchDateType, "search-date-type", "", "date field to search by")
	listCmd.Flags().StringVar(&listReq.Order, "order", "", "sort order")
	listCmd.Flags().StringVar(&listReq.Income, "income", "", "income selector")

	getSpec, _ := FindCommand("invoice", "get")
	var getReq invoice.GetRequest
	var includes []string
	var additionalFields []string
	getCmd := &cobra.Command{
		Use:   getSpec.Use,
		Short: getSpec.Short,
		Long:  BuildLongDescription(getSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, getSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice get"}, appErr)
			}
			getReq.ConfigPath = globals.Config
			getReq.Profile = globals.Profile
			getReq.Env = config.LookupEnv()
			getReq.Timeout = timeoutFromGlobals(globals)
			getReq.MaxRetries = globals.MaxRetries
			getReq.Includes = includes
			getReq.AdditionalFields = additionalFields

			start := time.Now()
			result, err := deps.Invoice.Get(cmd.Context(), getReq)
			meta := output.Meta{
				Command:    "invoice get",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          result.Invoice,
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	getCmd.Flags().StringVar(&getReq.ID, "id", "", "invoice ID")
	getCmd.Flags().StringSliceVar(&includes, "include", nil, "request upstream invoice includes such as descriptions")
	getCmd.Flags().StringSliceVar(&additionalFields, "additional-field", nil, "request additional upstream invoice fields such as cancel_reason or connected_payments")
	getCmd.Flags().StringVar(&getReq.CorrectionDetails, "correction-positions", "", "request correction position details such as full")
	_ = getCmd.MarkFlagRequired("id")

	downloadSpec, _ := FindCommand("invoice", "download")
	var downloadReq invoice.DownloadRequest
	downloadCmd := &cobra.Command{
		Use:   downloadSpec.Use,
		Short: downloadSpec.Short,
		Long:  BuildLongDescription(downloadSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, downloadSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice download"}, appErr)
			}
			downloadReq.ConfigPath = globals.Config
			downloadReq.Profile = globals.Profile
			downloadReq.Env = config.LookupEnv()
			downloadReq.Timeout = timeoutFromGlobals(globals)
			downloadReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Invoice.Download(cmd.Context(), downloadReq)
			meta := output.Meta{
				Command:    "invoice download",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:     result,
				Warnings: warnings,
				Meta:     meta,
				HumanRenderer: output.LinesRenderer{
					Lines: func(data any) ([]string, error) {
						res := data.(*invoice.DownloadResponse)
						return []string{res.Path}, nil
					},
				},
			})
		},
	}
	downloadCmd.Flags().StringVar(&downloadReq.ID, "id", "", "invoice ID")
	downloadCmd.Flags().StringVar(&downloadReq.Path, "path", "", "explicit output file path")
	downloadCmd.Flags().StringVar(&downloadReq.Dir, "dir", "", "output directory")
	downloadCmd.Flags().StringVar(&downloadReq.PrintOption, "print-option", "", "PDF print option")
	_ = downloadCmd.MarkFlagRequired("id")

	createSpec, _ := FindCommand("invoice", "create")
	var createInput string
	var createReq invoice.CreateRequest
	createCmd := &cobra.Command{
		Use:   createSpec.Use,
		Short: createSpec.Short,
		Long:  BuildLongDescription(createSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, createSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice create"}, appErr)
			}
			input, err := jsoninput.ParseObject(createInput, cmd.InOrStdin(), "invoice")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice create"}, err)
			}

			createReq.ConfigPath = globals.Config
			createReq.Profile = globals.Profile
			createReq.Env = config.LookupEnv()
			createReq.Timeout = timeoutFromGlobals(globals)
			createReq.MaxRetries = globals.MaxRetries
			createReq.Input = input
			createReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Invoice.Create(cmd.Context(), createReq)
			meta := output.Meta{
				Command:    "invoice create",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          invoiceCreateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	createCmd.Flags().StringVar(&createInput, "input", "", "invoice JSON input as inline JSON, @file, or - for stdin")
	createCmd.Flags().BoolVar(&createReq.IdentifyOSS, "identify-oss", false, "validate OSS eligibility before marking the invoice as OSS")
	createCmd.Flags().BoolVar(&createReq.FillDefaultDescriptions, "fill-default-descriptions", false, "include default account descriptions on the created invoice")
	createCmd.Flags().StringVar(&createReq.CorrectionPositions, "correction-positions", "", "pass a correction positions companion option such as full")
	_ = createCmd.MarkFlagRequired("input")

	updateSpec, _ := FindCommand("invoice", "update")
	var updateReq invoice.UpdateRequest
	var updateInput string
	updateCmd := &cobra.Command{
		Use:   updateSpec.Use,
		Short: updateSpec.Short,
		Long:  BuildLongDescription(updateSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, updateSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice update"}, appErr)
			}
			input, err := jsoninput.ParseObject(updateInput, cmd.InOrStdin(), "invoice")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice update"}, err)
			}

			updateReq.ConfigPath = globals.Config
			updateReq.Profile = globals.Profile
			updateReq.Env = config.LookupEnv()
			updateReq.Timeout = timeoutFromGlobals(globals)
			updateReq.MaxRetries = globals.MaxRetries
			updateReq.Input = input
			updateReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Invoice.Update(cmd.Context(), updateReq)
			meta := output.Meta{
				Command:    "invoice update",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          invoiceUpdateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	updateCmd.Flags().StringVar(&updateReq.ID, "id", "", "invoice ID")
	updateCmd.Flags().StringVar(&updateInput, "input", "", "invoice JSON input as inline JSON, @file, or - for stdin")
	updateCmd.Flags().BoolVar(&updateReq.IdentifyOSS, "identify-oss", false, "validate OSS eligibility before marking the invoice as OSS")
	updateCmd.Flags().BoolVar(&updateReq.FillDefaultDescriptions, "fill-default-descriptions", false, "include default account descriptions on the updated invoice")
	updateCmd.Flags().StringVar(&updateReq.CorrectionPositions, "correction-positions", "", "pass a correction positions companion option such as full")
	_ = updateCmd.MarkFlagRequired("id")
	_ = updateCmd.MarkFlagRequired("input")

	deleteSpec, _ := FindCommand("invoice", "delete")
	var deleteReq invoice.DeleteRequest
	var deleteYes bool
	deleteCmd := &cobra.Command{
		Use:   deleteSpec.Use,
		Short: deleteSpec.Short,
		Long:  BuildLongDescription(deleteSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, deleteSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice delete"}, appErr)
			}
			if !deleteYes {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice delete"}, output.Usage("confirmation_required", "--yes is required for invoice delete", "rerun with --yes to delete the invoice"))
			}

			deleteReq.ConfigPath = globals.Config
			deleteReq.Profile = globals.Profile
			deleteReq.Env = config.LookupEnv()
			deleteReq.Timeout = timeoutFromGlobals(globals)
			deleteReq.MaxRetries = globals.MaxRetries
			deleteReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Invoice.Delete(cmd.Context(), deleteReq)
			meta := output.Meta{
				Command:    "invoice delete",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          invoiceDeleteData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	deleteCmd.Flags().StringVar(&deleteReq.ID, "id", "", "invoice ID")
	deleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "confirm invoice deletion")
	_ = deleteCmd.MarkFlagRequired("id")

	sendSpec, _ := FindCommand("invoice", "send-email")
	var sendReq invoice.SendEmailRequest
	sendCmd := &cobra.Command{
		Use:   sendSpec.Use,
		Short: sendSpec.Short,
		Long:  BuildLongDescription(sendSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, sendSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice send-email"}, appErr)
			}
			sendReq.ConfigPath = globals.Config
			sendReq.Profile = globals.Profile
			sendReq.Env = config.LookupEnv()
			sendReq.Timeout = timeoutFromGlobals(globals)
			sendReq.MaxRetries = globals.MaxRetries
			sendReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Invoice.SendEmail(cmd.Context(), sendReq)
			meta := output.Meta{
				Command:    "invoice send-email",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          invoiceSendEmailData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	sendCmd.Flags().StringVar(&sendReq.ID, "id", "", "invoice ID")
	sendCmd.Flags().StringSliceVar(&sendReq.EmailTo, "email-to", nil, "override email recipients; may be repeated")
	sendCmd.Flags().StringSliceVar(&sendReq.EmailCC, "email-cc", nil, "override email CC recipients; may be repeated")
	sendCmd.Flags().BoolVar(&sendReq.EmailPDF, "email-pdf", false, "attach the invoice PDF to the email")
	sendCmd.Flags().BoolVar(&sendReq.UpdateBuyerEmail, "update-buyer-email", false, "update the invoice buyer or recipient email when email-to is provided")
	sendCmd.Flags().StringVar(&sendReq.PrintOption, "print-option", "", "PDF print option")
	_ = sendCmd.MarkFlagRequired("id")

	statusSpec, _ := FindCommand("invoice", "change-status")
	var statusReq invoice.ChangeStatusRequest
	statusCmd := &cobra.Command{
		Use:   statusSpec.Use,
		Short: statusSpec.Short,
		Long:  BuildLongDescription(statusSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, statusSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice change-status"}, appErr)
			}
			statusReq.ConfigPath = globals.Config
			statusReq.Profile = globals.Profile
			statusReq.Env = config.LookupEnv()
			statusReq.Timeout = timeoutFromGlobals(globals)
			statusReq.MaxRetries = globals.MaxRetries
			statusReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Invoice.ChangeStatus(cmd.Context(), statusReq)
			meta := output.Meta{
				Command:    "invoice change-status",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          invoiceChangeStatusData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	statusCmd.Flags().StringVar(&statusReq.ID, "id", "", "invoice ID")
	statusCmd.Flags().StringVar(&statusReq.Status, "status", "", "target invoice status")
	_ = statusCmd.MarkFlagRequired("id")
	_ = statusCmd.MarkFlagRequired("status")

	cancelSpec, _ := FindCommand("invoice", "cancel")
	var cancelReq invoice.CancelRequest
	var cancelYes bool
	cancelCmd := &cobra.Command{
		Use:   cancelSpec.Use,
		Short: cancelSpec.Short,
		Long:  BuildLongDescription(cancelSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, cancelSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice cancel"}, appErr)
			}
			if !cancelYes {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice cancel"}, output.Usage("confirmation_required", "--yes is required for invoice cancel", "rerun with --yes to cancel the invoice"))
			}
			cancelReq.ConfigPath = globals.Config
			cancelReq.Profile = globals.Profile
			cancelReq.Env = config.LookupEnv()
			cancelReq.Timeout = timeoutFromGlobals(globals)
			cancelReq.MaxRetries = globals.MaxRetries
			cancelReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Invoice.Cancel(cmd.Context(), cancelReq)
			meta := output.Meta{
				Command:    "invoice cancel",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          invoiceCancelData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	cancelCmd.Flags().StringVar(&cancelReq.ID, "id", "", "invoice ID")
	cancelCmd.Flags().StringVar(&cancelReq.Reason, "reason", "", "optional cancellation reason")
	cancelCmd.Flags().BoolVar(&cancelYes, "yes", false, "confirm invoice cancellation")
	_ = cancelCmd.MarkFlagRequired("id")

	publicLinkSpec, _ := FindCommand("invoice", "public-link")
	var publicLinkReq invoice.PublicLinkRequest
	publicLinkCmd := &cobra.Command{
		Use:   publicLinkSpec.Use,
		Short: publicLinkSpec.Short,
		Long:  BuildLongDescription(publicLinkSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, publicLinkSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice public-link"}, appErr)
			}
			publicLinkReq.ConfigPath = globals.Config
			publicLinkReq.Profile = globals.Profile
			publicLinkReq.Env = config.LookupEnv()
			publicLinkReq.Timeout = timeoutFromGlobals(globals)
			publicLinkReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Invoice.PublicLink(cmd.Context(), publicLinkReq)
			meta := output.Meta{
				Command:    "invoice public-link",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          result,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	publicLinkCmd.Flags().StringVar(&publicLinkReq.ID, "id", "", "invoice ID")
	_ = publicLinkCmd.MarkFlagRequired("id")

	addAttachmentSpec, _ := FindCommand("invoice", "add-attachment")
	var addAttachmentReq invoice.AddAttachmentRequest
	var attachmentFile string
	addAttachmentCmd := &cobra.Command{
		Use:   addAttachmentSpec.Use,
		Short: addAttachmentSpec.Short,
		Long:  BuildLongDescription(addAttachmentSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, addAttachmentSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice add-attachment"}, appErr)
			}
			name, data, err := readAttachmentInput(attachmentFile, addAttachmentReq.Name, cmd.InOrStdin())
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice add-attachment"}, err)
			}

			addAttachmentReq.ConfigPath = globals.Config
			addAttachmentReq.Profile = globals.Profile
			addAttachmentReq.Env = config.LookupEnv()
			addAttachmentReq.Timeout = timeoutFromGlobals(globals)
			addAttachmentReq.MaxRetries = globals.MaxRetries
			addAttachmentReq.Name = name
			addAttachmentReq.Content = data
			addAttachmentReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Invoice.AddAttachment(cmd.Context(), addAttachmentReq)
			meta := output.Meta{
				Command:    "invoice add-attachment",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          invoiceAddAttachmentData(result),
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	addAttachmentCmd.Flags().StringVar(&addAttachmentReq.ID, "id", "", "invoice ID")
	addAttachmentCmd.Flags().StringVar(&attachmentFile, "file", "", "attachment file path or - for stdin")
	addAttachmentCmd.Flags().StringVar(&addAttachmentReq.Name, "name", "", "attachment file name; required when --file - is used")
	_ = addAttachmentCmd.MarkFlagRequired("id")
	_ = addAttachmentCmd.MarkFlagRequired("file")

	downloadAttachmentsSpec, _ := FindCommand("invoice", "download-attachments")
	var downloadAttachmentsReq invoice.DownloadAttachmentsRequest
	downloadAttachmentsCmd := &cobra.Command{
		Use:   downloadAttachmentsSpec.Use,
		Short: downloadAttachmentsSpec.Short,
		Long:  BuildLongDescription(downloadAttachmentsSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, downloadAttachmentsSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice download-attachments"}, appErr)
			}
			downloadAttachmentsReq.ConfigPath = globals.Config
			downloadAttachmentsReq.Profile = globals.Profile
			downloadAttachmentsReq.Env = config.LookupEnv()
			downloadAttachmentsReq.Timeout = timeoutFromGlobals(globals)
			downloadAttachmentsReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Invoice.DownloadAttachments(cmd.Context(), downloadAttachmentsReq)
			meta := output.Meta{
				Command:    "invoice download-attachments",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:     result,
				Warnings: warnings,
				Meta:     meta,
				HumanRenderer: output.LinesRenderer{
					Lines: func(data any) ([]string, error) {
						res := data.(*invoice.DownloadAttachmentsResponse)
						return []string{res.Path}, nil
					},
				},
			})
		},
	}
	downloadAttachmentsCmd.Flags().StringVar(&downloadAttachmentsReq.ID, "id", "", "invoice ID")
	downloadAttachmentsCmd.Flags().StringVar(&downloadAttachmentsReq.Path, "path", "", "explicit output file path")
	downloadAttachmentsCmd.Flags().StringVar(&downloadAttachmentsReq.Dir, "dir", "", "output directory")
	_ = downloadAttachmentsCmd.MarkFlagRequired("id")

	fiscalPrintSpec, _ := FindCommand("invoice", "fiscal-print")
	var fiscalPrintReq invoice.FiscalPrintRequest
	fiscalPrintCmd := &cobra.Command{
		Use:   fiscalPrintSpec.Use,
		Short: fiscalPrintSpec.Short,
		Long:  BuildLongDescription(fiscalPrintSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, fiscalPrintSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "invoice fiscal-print"}, appErr)
			}
			fiscalPrintReq.ConfigPath = globals.Config
			fiscalPrintReq.Profile = globals.Profile
			fiscalPrintReq.Env = config.LookupEnv()
			fiscalPrintReq.Timeout = timeoutFromGlobals(globals)
			fiscalPrintReq.MaxRetries = globals.MaxRetries
			fiscalPrintReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Invoice.FiscalPrint(cmd.Context(), fiscalPrintReq)
			meta := output.Meta{
				Command:    "invoice fiscal-print",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          invoiceFiscalPrintData(result),
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	fiscalPrintCmd.Flags().StringSliceVar(&fiscalPrintReq.InvoiceIDs, "invoice-id", nil, "invoice ID to send to fiscal print; may be repeated")
	fiscalPrintCmd.Flags().StringVar(&fiscalPrintReq.Printer, "printer", "", "target fiscal printer name")

	invoiceCmd.AddCommand(
		listCmd,
		getCmd,
		downloadCmd,
		createCmd,
		updateCmd,
		deleteCmd,
		sendCmd,
		statusCmd,
		cancelCmd,
		publicLinkCmd,
		addAttachmentCmd,
		downloadAttachmentsCmd,
		fiscalPrintCmd,
	)
	return invoiceCmd
}

func newRecurringCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	recurringCmd := &cobra.Command{
		Use:   "recurring",
		Short: "Read and manage recurring invoice definitions",
	}

	listSpec, _ := FindCommand("recurring", "list")
	listCmd := &cobra.Command{
		Use:   listSpec.Use,
		Short: listSpec.Short,
		Long:  BuildLongDescription(listSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, listSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "recurring list"}, appErr)
			}

			start := time.Now()
			result, err := deps.Recurring.List(cmd.Context(), recurring.ListRequest{
				ConfigPath: globals.Config,
				Profile:    globals.Profile,
				Env:        config.LookupEnv(),
				Timeout:    timeoutFromGlobals(globals),
				MaxRetries: globals.MaxRetries,
			})
			meta := output.Meta{
				Command:    "recurring list",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:           result.Recurrings,
				RawBody:        result.RawBody,
				Warnings:       warnings,
				Meta:           meta,
				HumanRenderer:  output.TableRenderer{},
				DefaultColumns: defaultColumns(listSpec, []string{"id", "name", "invoice_id", "every", "next_invoice_date", "send_email"}),
			})
		},
	}

	createSpec, _ := FindCommand("recurring", "create")
	var createInput string
	createCmd := &cobra.Command{
		Use:   createSpec.Use,
		Short: createSpec.Short,
		Long:  BuildLongDescription(createSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, createSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "recurring create"}, appErr)
			}
			input, err := jsoninput.ParseObject(createInput, cmd.InOrStdin(), "recurring")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "recurring create"}, err)
			}

			start := time.Now()
			result, err := deps.Recurring.Create(cmd.Context(), recurring.CreateRequest{
				ConfigPath: globals.Config,
				Profile:    globals.Profile,
				Env:        config.LookupEnv(),
				Timeout:    timeoutFromGlobals(globals),
				MaxRetries: globals.MaxRetries,
				Input:      input,
				DryRun:     globals.DryRun,
			})
			meta := output.Meta{
				Command:    "recurring create",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          recurringCreateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	createCmd.Flags().StringVar(&createInput, "input", "", "recurring JSON input as inline JSON, @file, or - for stdin")
	_ = createCmd.MarkFlagRequired("input")

	updateSpec, _ := FindCommand("recurring", "update")
	var updateReq recurring.UpdateRequest
	var updateInput string
	updateCmd := &cobra.Command{
		Use:   updateSpec.Use,
		Short: updateSpec.Short,
		Long:  BuildLongDescription(updateSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, updateSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "recurring update"}, appErr)
			}
			input, err := jsoninput.ParseObject(updateInput, cmd.InOrStdin(), "recurring")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "recurring update"}, err)
			}

			updateReq.ConfigPath = globals.Config
			updateReq.Profile = globals.Profile
			updateReq.Env = config.LookupEnv()
			updateReq.Timeout = timeoutFromGlobals(globals)
			updateReq.MaxRetries = globals.MaxRetries
			updateReq.Input = input
			updateReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Recurring.Update(cmd.Context(), updateReq)
			meta := output.Meta{
				Command:    "recurring update",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          recurringUpdateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	updateCmd.Flags().StringVar(&updateReq.ID, "id", "", "recurring definition ID")
	updateCmd.Flags().StringVar(&updateInput, "input", "", "recurring JSON input as inline JSON, @file, or - for stdin")
	_ = updateCmd.MarkFlagRequired("id")
	_ = updateCmd.MarkFlagRequired("input")

	recurringCmd.AddCommand(listCmd, createCmd, updateCmd)
	return recurringCmd
}

func newProductCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	productCmd := &cobra.Command{
		Use:   "product",
		Short: "Read and manage products",
	}

	listSpec, _ := FindCommand("product", "list")
	listReq := product.ListRequest{Page: 1, PerPage: 25}
	listCmd := &cobra.Command{
		Use:   listSpec.Use,
		Short: listSpec.Short,
		Long:  BuildLongDescription(listSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, listSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "product list"}, appErr)
			}
			listReq.ConfigPath = globals.Config
			listReq.Profile = globals.Profile
			listReq.Env = config.LookupEnv()
			listReq.Timeout = timeoutFromGlobals(globals)
			listReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Product.List(cmd.Context(), listReq)
			meta := output.Meta{
				Command:    "product list",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
				meta.Pagination = &result.Pagination
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:           result.Products,
				RawBody:        result.RawBody,
				Warnings:       warnings,
				Meta:           meta,
				HumanRenderer:  output.TableRenderer{},
				DefaultColumns: defaultColumns(listSpec, []string{"id", "name", "code", "price_gross", "tax", "stock_level"}),
			})
		},
	}
	listCmd.Flags().IntVar(&listReq.Page, "page", 1, "requested result page")
	listCmd.Flags().IntVar(&listReq.PerPage, "per-page", 25, "requested result count per page")
	listCmd.Flags().StringVar(&listReq.DateFrom, "date-from", "", "filter products added or changed since a date such as 2025-11-01")
	listCmd.Flags().StringVar(&listReq.WarehouseID, "warehouse-id", "", "show stock levels for a specific warehouse")

	getSpec, _ := FindCommand("product", "get")
	var getReq product.GetRequest
	getCmd := &cobra.Command{
		Use:   getSpec.Use,
		Short: getSpec.Short,
		Long:  BuildLongDescription(getSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, getSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "product get"}, appErr)
			}
			getReq.ConfigPath = globals.Config
			getReq.Profile = globals.Profile
			getReq.Env = config.LookupEnv()
			getReq.Timeout = timeoutFromGlobals(globals)
			getReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Product.Get(cmd.Context(), getReq)
			meta := output.Meta{
				Command:    "product get",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          result.Product,
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	getCmd.Flags().StringVar(&getReq.ID, "id", "", "product ID")
	getCmd.Flags().StringVar(&getReq.WarehouseID, "warehouse-id", "", "show stock level for a specific warehouse")
	_ = getCmd.MarkFlagRequired("id")

	createSpec, _ := FindCommand("product", "create")
	var createInput string
	createCmd := &cobra.Command{
		Use:   createSpec.Use,
		Short: createSpec.Short,
		Long:  BuildLongDescription(createSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, createSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "product create"}, appErr)
			}
			input, err := jsoninput.ParseObject(createInput, cmd.InOrStdin(), "product")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "product create"}, err)
			}

			start := time.Now()
			result, err := deps.Product.Create(cmd.Context(), product.CreateRequest{
				ConfigPath: globals.Config,
				Profile:    globals.Profile,
				Env:        config.LookupEnv(),
				Timeout:    timeoutFromGlobals(globals),
				MaxRetries: globals.MaxRetries,
				Input:      input,
				DryRun:     globals.DryRun,
			})
			meta := output.Meta{
				Command:    "product create",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          productCreateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	createCmd.Flags().StringVar(&createInput, "input", "", "product JSON input as inline JSON, @file, or - for stdin")
	_ = createCmd.MarkFlagRequired("input")

	updateSpec, _ := FindCommand("product", "update")
	var updateReq product.UpdateRequest
	var updateInput string
	updateCmd := &cobra.Command{
		Use:   updateSpec.Use,
		Short: updateSpec.Short,
		Long:  BuildLongDescription(updateSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, updateSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "product update"}, appErr)
			}
			input, err := jsoninput.ParseObject(updateInput, cmd.InOrStdin(), "product")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "product update"}, err)
			}

			updateReq.ConfigPath = globals.Config
			updateReq.Profile = globals.Profile
			updateReq.Env = config.LookupEnv()
			updateReq.Timeout = timeoutFromGlobals(globals)
			updateReq.MaxRetries = globals.MaxRetries
			updateReq.Input = input
			updateReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Product.Update(cmd.Context(), updateReq)
			meta := output.Meta{
				Command:    "product update",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          productUpdateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	updateCmd.Flags().StringVar(&updateReq.ID, "id", "", "product ID")
	updateCmd.Flags().StringVar(&updateInput, "input", "", "product JSON input as inline JSON, @file, or - for stdin")
	_ = updateCmd.MarkFlagRequired("id")
	_ = updateCmd.MarkFlagRequired("input")

	productCmd.AddCommand(listCmd, getCmd, createCmd, updateCmd)
	return productCmd
}

func newPriceListCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	priceListCmd := &cobra.Command{
		Use:   "price-list",
		Short: "Read and manage price lists",
	}

	listSpec, _ := FindCommand("price-list", "list")
	listReq := pricelist.ListRequest{Page: 1, PerPage: 25}
	listCmd := &cobra.Command{
		Use:   listSpec.Use,
		Short: listSpec.Short,
		Long:  BuildLongDescription(listSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, listSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "price-list list"}, appErr)
			}
			listReq.ConfigPath = globals.Config
			listReq.Profile = globals.Profile
			listReq.Env = config.LookupEnv()
			listReq.Timeout = timeoutFromGlobals(globals)
			listReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.PriceList.List(cmd.Context(), listReq)
			meta := output.Meta{
				Command:    "price-list list",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
				meta.Pagination = &result.Pagination
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:           result.PriceLists,
				RawBody:        result.RawBody,
				Warnings:       warnings,
				Meta:           meta,
				HumanRenderer:  output.TableRenderer{},
				DefaultColumns: defaultColumns(listSpec, []string{"id", "name", "currency", "description"}),
			})
		},
	}
	listCmd.Flags().IntVar(&listReq.Page, "page", 1, "requested result page")
	listCmd.Flags().IntVar(&listReq.PerPage, "per-page", 25, "requested result count per page")

	getSpec, _ := FindCommand("price-list", "get")
	var getReq pricelist.GetRequest
	getCmd := &cobra.Command{
		Use:   getSpec.Use,
		Short: getSpec.Short,
		Long:  BuildLongDescription(getSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, getSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "price-list get"}, appErr)
			}
			getReq.ConfigPath = globals.Config
			getReq.Profile = globals.Profile
			getReq.Env = config.LookupEnv()
			getReq.Timeout = timeoutFromGlobals(globals)
			getReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.PriceList.Get(cmd.Context(), getReq)
			meta := output.Meta{
				Command:    "price-list get",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          result.PriceList,
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	getCmd.Flags().StringVar(&getReq.ID, "id", "", "price list ID")
	_ = getCmd.MarkFlagRequired("id")

	createSpec, _ := FindCommand("price-list", "create")
	var createInput string
	createCmd := &cobra.Command{
		Use:   createSpec.Use,
		Short: createSpec.Short,
		Long:  BuildLongDescription(createSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, createSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "price-list create"}, appErr)
			}
			input, err := jsoninput.ParseObject(createInput, cmd.InOrStdin(), "price list")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "price-list create"}, err)
			}

			start := time.Now()
			result, err := deps.PriceList.Create(cmd.Context(), pricelist.CreateRequest{
				ConfigPath: globals.Config,
				Profile:    globals.Profile,
				Env:        config.LookupEnv(),
				Timeout:    timeoutFromGlobals(globals),
				MaxRetries: globals.MaxRetries,
				Input:      input,
				DryRun:     globals.DryRun,
			})
			meta := output.Meta{
				Command:    "price-list create",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          priceListCreateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	createCmd.Flags().StringVar(&createInput, "input", "", "price list JSON input as inline JSON, @file, or - for stdin")
	_ = createCmd.MarkFlagRequired("input")

	updateSpec, _ := FindCommand("price-list", "update")
	var updateReq pricelist.UpdateRequest
	var updateInput string
	updateCmd := &cobra.Command{
		Use:   updateSpec.Use,
		Short: updateSpec.Short,
		Long:  BuildLongDescription(updateSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, updateSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "price-list update"}, appErr)
			}
			input, err := jsoninput.ParseObject(updateInput, cmd.InOrStdin(), "price list")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "price-list update"}, err)
			}

			updateReq.ConfigPath = globals.Config
			updateReq.Profile = globals.Profile
			updateReq.Env = config.LookupEnv()
			updateReq.Timeout = timeoutFromGlobals(globals)
			updateReq.MaxRetries = globals.MaxRetries
			updateReq.Input = input
			updateReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.PriceList.Update(cmd.Context(), updateReq)
			meta := output.Meta{
				Command:    "price-list update",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          priceListUpdateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	updateCmd.Flags().StringVar(&updateReq.ID, "id", "", "price list ID")
	updateCmd.Flags().StringVar(&updateInput, "input", "", "price list JSON input as inline JSON, @file, or - for stdin")
	_ = updateCmd.MarkFlagRequired("id")
	_ = updateCmd.MarkFlagRequired("input")

	deleteSpec, _ := FindCommand("price-list", "delete")
	var deleteReq pricelist.DeleteRequest
	var deleteYes bool
	deleteCmd := &cobra.Command{
		Use:   deleteSpec.Use,
		Short: deleteSpec.Short,
		Long:  BuildLongDescription(deleteSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, deleteSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "price-list delete"}, appErr)
			}
			if !deleteYes {
				return writeCommandError(cmd, opts, output.Meta{Command: "price-list delete"}, output.Usage("confirmation_required", "--yes is required for price-list delete", "rerun with --yes to delete the price list"))
			}

			deleteReq.ConfigPath = globals.Config
			deleteReq.Profile = globals.Profile
			deleteReq.Env = config.LookupEnv()
			deleteReq.Timeout = timeoutFromGlobals(globals)
			deleteReq.MaxRetries = globals.MaxRetries
			deleteReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.PriceList.Delete(cmd.Context(), deleteReq)
			meta := output.Meta{
				Command:    "price-list delete",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)

			humanRenderer := output.HumanRenderer(output.LinesRenderer{
				Lines: func(data any) ([]string, error) {
					res := data.(*pricelist.DeleteResponse)
					return []string{fmt.Sprintf("deleted price list %s", res.ID)}, nil
				},
			})
			data := priceListDeleteData(result)
			if result.DryRun != nil {
				humanRenderer = output.JSONRenderer{}
			}
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          data,
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: humanRenderer,
			})
		},
	}
	deleteCmd.Flags().StringVar(&deleteReq.ID, "id", "", "price list ID")
	deleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "confirm price list deletion")
	_ = deleteCmd.MarkFlagRequired("id")

	priceListCmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, deleteCmd)
	return priceListCmd
}

func newWarehouseDocumentCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	warehouseCmd := &cobra.Command{
		Use:   "warehouse-document",
		Short: "Read and manage warehouse documents",
	}

	listSpec, _ := FindCommand("warehouse-document", "list")
	listReq := warehousedocument.ListRequest{Page: 1, PerPage: 25}
	listCmd := &cobra.Command{
		Use:   listSpec.Use,
		Short: listSpec.Short,
		Long:  BuildLongDescription(listSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, listSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "warehouse-document list"}, appErr)
			}
			listReq.ConfigPath = globals.Config
			listReq.Profile = globals.Profile
			listReq.Env = config.LookupEnv()
			listReq.Timeout = timeoutFromGlobals(globals)
			listReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Warehouse.List(cmd.Context(), listReq)
			meta := output.Meta{
				Command:    "warehouse-document list",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
				meta.Pagination = &result.Pagination
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:           result.WarehouseDocuments,
				RawBody:        result.RawBody,
				Warnings:       warnings,
				Meta:           meta,
				HumanRenderer:  output.TableRenderer{},
				DefaultColumns: defaultColumns(listSpec, []string{"id", "kind", "number", "issue_date", "warehouse_id", "client_name"}),
			})
		},
	}
	listCmd.Flags().IntVar(&listReq.Page, "page", 1, "requested result page")
	listCmd.Flags().IntVar(&listReq.PerPage, "per-page", 25, "requested result count per page")

	getSpec, _ := FindCommand("warehouse-document", "get")
	var getReq warehousedocument.GetRequest
	getCmd := &cobra.Command{
		Use:   getSpec.Use,
		Short: getSpec.Short,
		Long:  BuildLongDescription(getSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, getSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "warehouse-document get"}, appErr)
			}
			getReq.ConfigPath = globals.Config
			getReq.Profile = globals.Profile
			getReq.Env = config.LookupEnv()
			getReq.Timeout = timeoutFromGlobals(globals)
			getReq.MaxRetries = globals.MaxRetries

			start := time.Now()
			result, err := deps.Warehouse.Get(cmd.Context(), getReq)
			meta := output.Meta{
				Command:    "warehouse-document get",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          result.WarehouseDocument,
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	getCmd.Flags().StringVar(&getReq.ID, "id", "", "warehouse document ID")
	_ = getCmd.MarkFlagRequired("id")

	createSpec, _ := FindCommand("warehouse-document", "create")
	var createInput string
	createCmd := &cobra.Command{
		Use:   createSpec.Use,
		Short: createSpec.Short,
		Long:  BuildLongDescription(createSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, createSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "warehouse-document create"}, appErr)
			}
			input, err := jsoninput.ParseObject(createInput, cmd.InOrStdin(), "warehouse document")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "warehouse-document create"}, err)
			}

			start := time.Now()
			result, err := deps.Warehouse.Create(cmd.Context(), warehousedocument.CreateRequest{
				ConfigPath: globals.Config,
				Profile:    globals.Profile,
				Env:        config.LookupEnv(),
				Timeout:    timeoutFromGlobals(globals),
				MaxRetries: globals.MaxRetries,
				Input:      input,
				DryRun:     globals.DryRun,
			})
			meta := output.Meta{
				Command:    "warehouse-document create",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          warehouseDocumentCreateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	createCmd.Flags().StringVar(&createInput, "input", "", "warehouse document JSON input as inline JSON, @file, or - for stdin")
	_ = createCmd.MarkFlagRequired("input")

	updateSpec, _ := FindCommand("warehouse-document", "update")
	var updateReq warehousedocument.UpdateRequest
	var updateInput string
	updateCmd := &cobra.Command{
		Use:   updateSpec.Use,
		Short: updateSpec.Short,
		Long:  BuildLongDescription(updateSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, updateSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "warehouse-document update"}, appErr)
			}
			input, err := jsoninput.ParseObject(updateInput, cmd.InOrStdin(), "warehouse document")
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "warehouse-document update"}, err)
			}

			updateReq.ConfigPath = globals.Config
			updateReq.Profile = globals.Profile
			updateReq.Env = config.LookupEnv()
			updateReq.Timeout = timeoutFromGlobals(globals)
			updateReq.MaxRetries = globals.MaxRetries
			updateReq.Input = input
			updateReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Warehouse.Update(cmd.Context(), updateReq)
			meta := output.Meta{
				Command:    "warehouse-document update",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          warehouseDocumentUpdateData(result),
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	updateCmd.Flags().StringVar(&updateReq.ID, "id", "", "warehouse document ID")
	updateCmd.Flags().StringVar(&updateInput, "input", "", "warehouse document JSON input as inline JSON, @file, or - for stdin")
	_ = updateCmd.MarkFlagRequired("id")
	_ = updateCmd.MarkFlagRequired("input")

	deleteSpec, _ := FindCommand("warehouse-document", "delete")
	var deleteReq warehousedocument.DeleteRequest
	var deleteYes bool
	deleteCmd := &cobra.Command{
		Use:   deleteSpec.Use,
		Short: deleteSpec.Short,
		Long:  BuildLongDescription(deleteSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, deleteSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "warehouse-document delete"}, appErr)
			}
			if !deleteYes {
				return writeCommandError(cmd, opts, output.Meta{Command: "warehouse-document delete"}, output.Usage("confirmation_required", "--yes is required for warehouse-document delete", "rerun with --yes to delete the warehouse document"))
			}

			deleteReq.ConfigPath = globals.Config
			deleteReq.Profile = globals.Profile
			deleteReq.Env = config.LookupEnv()
			deleteReq.Timeout = timeoutFromGlobals(globals)
			deleteReq.MaxRetries = globals.MaxRetries
			deleteReq.DryRun = globals.DryRun

			start := time.Now()
			result, err := deps.Warehouse.Delete(cmd.Context(), deleteReq)
			meta := output.Meta{
				Command:    "warehouse-document delete",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if result != nil {
				meta.RequestID = result.RequestID
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)

			humanRenderer := output.HumanRenderer(output.LinesRenderer{
				Lines: func(data any) ([]string, error) {
					res := data.(*warehousedocument.DeleteResponse)
					return []string{fmt.Sprintf("deleted warehouse document %s", res.ID)}, nil
				},
			})
			data := warehouseDocumentDeleteData(result)
			if result.DryRun != nil {
				humanRenderer = output.JSONRenderer{}
			}
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          data,
				RawBody:       result.RawBody,
				Warnings:      warnings,
				Meta:          meta,
				HumanRenderer: humanRenderer,
			})
		},
	}
	deleteCmd.Flags().StringVar(&deleteReq.ID, "id", "", "warehouse document ID")
	deleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "confirm warehouse document deletion")
	_ = deleteCmd.MarkFlagRequired("id")

	warehouseCmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, deleteCmd)
	return warehouseCmd
}

func newDoctorCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	doctorSpec, _ := FindCommand("doctor", "run")
	var checkReleaseIntegrity bool
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate local configuration and API reachability",
	}
	runCmd := &cobra.Command{
		Use:   doctorSpec.Use,
		Short: doctorSpec.Short,
		Long:  BuildLongDescription(doctorSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, doctorSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "doctor run"}, appErr)
			}
			start := time.Now()
			result, err := deps.Doctor.Run(cmd.Context(), doctor.RunRequest{
				ConfigPath:            globals.Config,
				Profile:               globals.Profile,
				Env:                   config.LookupEnv(),
				Timeout:               timeoutFromGlobals(globals),
				MaxRetries:            globals.MaxRetries,
				Version:               Version,
				CheckReleaseIntegrity: checkReleaseIntegrity,
			})
			meta := output.Meta{
				Command:    "doctor run",
				Profile:    resultProfile(result),
				DurationMS: time.Since(start).Milliseconds(),
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:     result.Report,
				Warnings: append(append([]output.WarningDetail{}, warnings...), result.Warnings...),
				Meta:     meta,
				HumanRenderer: output.LinesRenderer{
					Lines: func(data any) ([]string, error) {
						report := data.(doctor.Report)
						lines := []string{fmt.Sprintf("status: %s", report.Status)}
						for _, check := range report.Checks {
							lines = append(lines, fmt.Sprintf("%s: %s - %s", check.Name, check.Status, check.Message))
						}
						return lines, nil
					},
				},
			})
		},
	}
	runCmd.Flags().BoolVar(&checkReleaseIntegrity, "check-release-integrity", false, "verify the running binary against published release metadata")
	doctorCmd.AddCommand(runCmd)
	return doctorCmd
}

func newSelfCommand(deps Dependencies, globals *globalOptions) *cobra.Command {
	selfCmd := &cobra.Command{
		Use:   "self",
		Short: "Maintain the CLI binary",
	}

	updateSpec, _ := FindCommand("self", "update")
	var targetVersion string
	updateCmd := &cobra.Command{
		Use:   updateSpec.Use,
		Short: updateSpec.Short,
		Long:  BuildLongDescription(updateSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, updateSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "self update"}, appErr)
			}

			start := time.Now()
			result, err := deps.Self.Update(cmd.Context(), selfupdate.UpdateRequest{
				CurrentVersion: Version,
				TargetVersion:  targetVersion,
				Timeout:        timeoutFromGlobals(globals),
				DryRun:         globals.DryRun,
			})
			meta := output.Meta{
				Command:    "self update",
				DurationMS: time.Since(start).Milliseconds(),
			}
			if err != nil {
				return writeCommandError(cmd, opts, meta, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:     result,
				Warnings: warnings,
				Meta:     meta,
				HumanRenderer: output.LinesRenderer{
					Lines: func(data any) ([]string, error) {
						res := data.(*selfupdate.UpdateResult)
						switch {
						case res.DryRun:
							return []string{
								fmt.Sprintf("would install %s to %s", res.TargetVersion, res.ExecutablePath),
								fmt.Sprintf("download: %s", res.DownloadURL),
							}, nil
						case res.AlreadyCurrent:
							return []string{
								fmt.Sprintf("already on %s at %s", res.TargetVersion, res.ExecutablePath),
							}, nil
						default:
							return []string{
								fmt.Sprintf("installed %s to %s", res.TargetVersion, res.ExecutablePath),
								fmt.Sprintf("previous version: %s", res.CurrentVersion),
							}, nil
						}
					},
				},
			})
		},
	}
	updateCmd.Flags().StringVar(&targetVersion, "version", "", "release tag to install, or latest when omitted")
	selfCmd.AddCommand(updateCmd)
	return selfCmd
}

func newSchemaCommand(_ Dependencies, globals *globalOptions) *cobra.Command {
	listSpec, _ := FindCommand("schema", "list")
	describeSpec, _ := FindCommand("schema", "<noun> <verb>")

	schemaCmd := &cobra.Command{
		Use:   "schema",
		Short: "Describe commands and schemas",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, describeSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "schema"}, appErr)
			}
			if len(args) != 2 {
				return writeCommandError(cmd, opts, output.Meta{Command: "schema"}, output.Usage("invalid_args", "schema requires either `list` or `<noun> <verb>`", "use `fakturownia schema list` or `fakturownia schema invoice list`"))
			}
			target, ok := FindCommand(args[0], args[1])
			if !ok {
				return writeCommandError(cmd, opts, output.Meta{Command: "schema " + strings.Join(args, " ")}, output.NotFound("command_not_found", fmt.Sprintf("command %s %s was not found", args[0], args[1]), "use `fakturownia schema list` to inspect the supported surface"))
			}
			schema, err := BuildCommandSchema(target)
			if err != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "schema " + strings.Join(args, " ")}, err)
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:          schema,
				Warnings:      warnings,
				Meta:          output.Meta{Command: "schema " + strings.Join(args, " "), DurationMS: 0},
				HumanRenderer: output.JSONRenderer{},
			})
		},
	}
	schemaCmd.Long = BuildLongDescription(describeSpec)

	listCmd := &cobra.Command{
		Use:   listSpec.Use,
		Short: listSpec.Short,
		Long:  BuildLongDescription(listSpec),
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, warnings, appErr := prepareOutputOptions(cmd, listSpec, globals)
			if appErr != nil {
				return writeCommandError(cmd, opts, output.Meta{Command: "schema list"}, appErr)
			}
			data := SchemaSummaries()
			rows := make([]map[string]any, 0, len(data))
			for _, item := range data {
				rows = append(rows, map[string]any{
					"noun":    item.Noun,
					"verb":    item.Verb,
					"use":     item.Use,
					"summary": item.Summary,
				})
			}
			writeWarnings(cmd.ErrOrStderr(), opts, warnings)
			return output.RenderSuccess(cmd.OutOrStdout(), opts, output.Result{
				Data:           rows,
				Warnings:       warnings,
				Meta:           output.Meta{Command: "schema list", DurationMS: 0},
				HumanRenderer:  output.TableRenderer{},
				DefaultColumns: []string{"noun", "verb", "summary"},
			})
		},
	}
	schemaCmd.AddCommand(listCmd)
	return schemaCmd
}

func prepareOutputOptions(cmd *cobra.Command, spec CommandSpec, globals *globalOptions) (output.Options, []output.WarningDetail, *output.AppError) {
	format := globals.Output
	if globals.JSON {
		format = "json"
	}
	format = strings.TrimSpace(format)
	if format == "" {
		format = "human"
	}
	if format != "human" && format != "json" {
		return output.Options{}, nil, output.Usage("invalid_output", fmt.Sprintf("unsupported output mode %q", format), "use --output human or --output json")
	}
	fields := trimValues(globals.Fields)
	columns := trimValues(globals.Columns)
	opts := output.Options{
		Format:  format,
		Raw:     globals.Raw,
		Quiet:   globals.Quiet,
		Fields:  fields,
		Columns: columns,
	}

	if globals.Raw {
		if !spec.RawSupported {
			return output.Options{}, nil, output.Usage("raw_unsupported", fmt.Sprintf("--raw is not supported for %s %s", spec.Noun, spec.Verb), "use --json for the structured CLI envelope")
		}
		if cmd.Flags().Changed("json") || cmd.Flags().Changed("output") || cmd.Flags().Changed("fields") || cmd.Flags().Changed("columns") || cmd.Flags().Changed("quiet") {
			return output.Options{}, nil, output.Usage("raw_conflict", "--raw cannot be combined with --json, --output, --fields, --columns, or --quiet", "drop the other output flags when using --raw")
		}
	}
	if globals.DryRun && spec.Mutating {
		if globals.Raw {
			return output.Options{}, nil, output.Usage("dry_run_raw_conflict", "--dry-run cannot be combined with --raw for mutating commands", "use --json or human output to inspect the planned request")
		}
		if len(opts.Fields) > 0 || len(opts.Columns) > 0 || opts.Quiet {
			return output.Options{}, nil, output.Usage("dry_run_output_conflict", "--dry-run for mutating commands cannot be combined with --fields, --columns, or --quiet", "inspect the full planned request instead")
		}
	}
	if opts.Format == "json" && opts.Quiet {
		return output.Options{}, nil, output.Usage("quiet_json_conflict", "--quiet cannot be combined with JSON output", "use --fields with --json or use --quiet with human output")
	}
	if opts.Format == "json" && len(opts.Columns) > 0 {
		return output.Options{}, nil, output.Usage("columns_json_conflict", "--columns only applies to human table output", "use --fields for JSON projection")
	}
	warnings, appErr := validateOutputSelection(spec, fields, columns)
	if appErr != nil {
		return output.Options{}, nil, appErr
	}
	return opts, warnings, nil
}

func timeoutFromGlobals(globals *globalOptions) time.Duration {
	if globals.TimeoutMS <= 0 {
		return 30 * time.Second
	}
	return time.Duration(globals.TimeoutMS) * time.Millisecond
}

func writeCommandError(cmd *cobra.Command, opts output.Options, meta output.Meta, err error) error {
	appErr := output.AsAppError(err)
	if renderErr := output.RenderError(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts, meta, appErr); renderErr != nil {
		return output.ExitError{Code: 9}
	}
	return output.ExitError{Code: appErr.ExitCode()}
}

func trimValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func writeWarnings(stderr io.Writer, opts output.Options, warnings []output.WarningDetail) {
	if opts.Format == "json" || len(warnings) == 0 {
		return
	}
	for _, warning := range warnings {
		fmt.Fprintf(stderr, "warning: %s (%s)\n", warning.Message, warning.Code)
	}
}

func defaultColumns(spec CommandSpec, fallback []string) []string {
	if spec.Output != nil && len(spec.Output.DefaultColumns) > 0 {
		return append([]string{}, spec.Output.DefaultColumns...)
	}
	return fallback
}

func clientCreateData(result *client.CreateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.Client
}

func clientUpdateData(result *client.UpdateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.Client
}

func clientDeleteData(result *client.DeleteResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result
}

func productCreateData(result *product.CreateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.Product
}

func productUpdateData(result *product.UpdateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.Product
}

func priceListCreateData(result *pricelist.CreateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.PriceList
}

func priceListUpdateData(result *pricelist.UpdateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.PriceList
}

func priceListDeleteData(result *pricelist.DeleteResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result
}

func invoiceCreateData(result *invoice.CreateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.Invoice
}

func invoiceUpdateData(result *invoice.UpdateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.Invoice
}

func invoiceDeleteData(result *invoice.DeleteResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	if result.Response != nil {
		return result.Response
	}
	return result
}

func invoiceSendEmailData(result *invoice.SendEmailResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	if result.Response != nil {
		return result.Response
	}
	return result
}

func invoiceChangeStatusData(result *invoice.ChangeStatusResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	if result.Response != nil {
		return result.Response
	}
	return result
}

func invoiceCancelData(result *invoice.CancelResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	if result.Response != nil {
		return result.Response
	}
	return result
}

func invoiceAddAttachmentData(result *invoice.AddAttachmentResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result
}

func invoiceFiscalPrintData(result *invoice.FiscalPrintResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result
}

func recurringCreateData(result *recurring.CreateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.Recurring
}

func recurringUpdateData(result *recurring.UpdateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.Recurring
}

func warehouseDocumentCreateData(result *warehousedocument.CreateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.WarehouseDocument
}

func warehouseDocumentUpdateData(result *warehousedocument.UpdateResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result.WarehouseDocument
}

func warehouseDocumentDeleteData(result *warehousedocument.DeleteResponse) any {
	if result == nil {
		return nil
	}
	if result.DryRun != nil {
		return result.DryRun
	}
	return result
}

func readAttachmentInput(source, explicitName string, stdin io.Reader) (string, []byte, error) {
	trimmedSource := strings.TrimSpace(source)
	if trimmedSource == "" {
		return "", nil, output.Usage("missing_file", "attachment file is required", "pass --file /path/to/file or --file - for stdin")
	}
	name := strings.TrimSpace(explicitName)
	switch trimmedSource {
	case "-":
		if name == "" {
			return "", nil, output.Usage("missing_name", "--name is required when --file - is used", "pass --name <file-name.ext> when reading attachment bytes from stdin")
		}
		if stdin == nil {
			return "", nil, output.Internal(nil, "stdin is not available")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", nil, output.Internal(err, "read attachment bytes from stdin")
		}
		if len(data) == 0 {
			return "", nil, output.Usage("empty_file", "attachment input cannot be empty", "provide attachment bytes through stdin")
		}
		return name, data, nil
	default:
		data, err := os.ReadFile(trimmedSource)
		if err != nil {
			return "", nil, output.Internal(err, "read attachment file")
		}
		if name == "" {
			name = filepath.Base(trimmedSource)
		}
		if len(data) == 0 {
			return "", nil, output.Usage("empty_file", "attachment input cannot be empty", "provide a non-empty file")
		}
		return name, data, nil
	}
}

func resultProfile(result any) string {
	switch typed := result.(type) {
	case interface{ GetProfile() string }:
		return typed.GetProfile()
	default:
		return ""
	}
}
