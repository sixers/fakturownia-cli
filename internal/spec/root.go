package spec

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sixers/fakturownia-cli/internal/auth"
	"github.com/sixers/fakturownia-cli/internal/client"
	"github.com/sixers/fakturownia-cli/internal/config"
	"github.com/sixers/fakturownia-cli/internal/doctor"
	"github.com/sixers/fakturownia-cli/internal/invoice"
	"github.com/sixers/fakturownia-cli/internal/output"
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
}

type ClientService interface {
	List(context.Context, client.ListRequest) (*client.ListResponse, error)
	Get(context.Context, client.GetRequest) (*client.GetResponse, error)
	Create(context.Context, client.CreateRequest) (*client.CreateResponse, error)
	Update(context.Context, client.UpdateRequest) (*client.UpdateResponse, error)
	Delete(context.Context, client.DeleteRequest) (*client.DeleteResponse, error)
}

type DoctorService interface {
	Run(context.Context, doctor.RunRequest) (*doctor.RunResult, error)
}

type Dependencies struct {
	Auth    AuthService
	Client  ClientService
	Invoice InvoiceService
	Doctor  DoctorService
	Stdout  io.Writer
	Stderr  io.Writer
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
	root.AddCommand(newDoctorCommand(deps, &globals))
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
		Short: "Read invoice data and PDF files",
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

	invoiceCmd.AddCommand(listCmd, getCmd, downloadCmd)
	return invoiceCmd
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

func resultProfile(result any) string {
	switch typed := result.(type) {
	case interface{ GetProfile() string }:
		return typed.GetProfile()
	case *auth.LoginResult:
		if typed == nil {
			return ""
		}
		return typed.Profile
	case *auth.StatusResult:
		if typed == nil {
			return ""
		}
		return typed.Profile
	case *auth.LogoutResult:
		if typed == nil {
			return ""
		}
		return typed.Profile
	case *invoice.ListResponse:
		if typed == nil {
			return ""
		}
		return typed.Profile
	case *invoice.GetResponse:
		if typed == nil {
			return ""
		}
		return typed.Profile
	case *invoice.DownloadResponse:
		if typed == nil {
			return ""
		}
		return typed.Profile
	case *doctor.RunResult:
		if typed == nil {
			return ""
		}
		return typed.Profile
	default:
		return ""
	}
}
