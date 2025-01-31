package cmd

import (
	"errors"

	env "github.com/Netflix/go-env"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/supabase/cli/internal/gen/keys"
	"github.com/supabase/cli/internal/gen/types/typescript"
	"github.com/supabase/cli/internal/utils"
	"github.com/supabase/cli/internal/utils/flags"
)

var (
	genCmd = &cobra.Command{
		GroupID: groupManagementAPI,
		Use:     "gen",
		Short:   "Run code generation tools",
	}

	keyNames  keys.CustomName
	keyOutput = utils.EnumFlag{
		Allowed: []string{
			utils.OutputEnv,
			utils.OutputJson,
			utils.OutputToml,
			utils.OutputYaml,
		},
		Value: utils.OutputEnv,
	}

	genKeysCmd = &cobra.Command{
		Use:   "keys",
		Short: "Generate keys for preview branch",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			es, err := env.EnvironToEnvSet(override)
			if err != nil {
				return err
			}
			if err := env.Unmarshal(es, &keyNames); err != nil {
				return err
			}
			return cmd.Root().PersistentPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return keys.Run(cmd.Context(), flags.ProjectRef, keyOutput.Value, keyNames, afero.NewOsFs())
		},
	}

	genTypesCmd = &cobra.Command{
		Use:   "types",
		Short: "Generate types from Postgres schema",
	}

	postgrestV9Compat bool

	genTypesTypescriptCmd = &cobra.Command{
		Use:   "typescript",
		Short: "Generate types for TypeScript",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if postgrestV9Compat && !cmd.Flags().Changed("db-url") {
				return errors.New("--postgrest-v9-compat can only be used together with --db-url.")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if flags.DbConfig.Host == "" {
				// If no flag is specified, prompt for project id.
				if err := flags.ParseProjectRef(ctx, afero.NewMemMapFs()); errors.Is(err, utils.ErrNotLinked) {
					return errors.New("Must specify one of --local, --linked, --project-id, or --db-url")
				} else if err != nil {
					return err
				}
			}
			return typescript.Run(ctx, flags.ProjectRef, flags.DbConfig, schema, postgrestV9Compat, afero.NewOsFs())
		},
		Example: `  supabase gen types typescript --local
  supabase gen types typescript --linked
  supabase gen types typescript --project-id abc-def-123 --schema public --schema private
  supabase gen types typescript --db-url 'postgresql://...' --schema public --schema auth`,
	}
)

func init() {
	genFlags := genTypesTypescriptCmd.Flags()
	genFlags.Bool("local", false, "Generate types from the local dev database.")
	genFlags.Bool("linked", false, "Generate types from the linked project.")
	genFlags.String("db-url", "", "Generate types from a database url.")
	genFlags.StringVar(&flags.ProjectRef, "project-id", "", "Generate types from a project ID.")
	genTypesTypescriptCmd.MarkFlagsMutuallyExclusive("local", "linked", "project-id", "db-url")
	genFlags.StringSliceVarP(&schema, "schema", "s", []string{}, "Comma separated list of schema to include.")
	genFlags.BoolVar(&postgrestV9Compat, "postgrest-v9-compat", false, "Generate types compatible with PostgREST v9 and below. Only use together with --db-url.")
	genTypesCmd.AddCommand(genTypesTypescriptCmd)
	genCmd.AddCommand(genTypesCmd)
	keyFlags := genKeysCmd.Flags()
	keyFlags.StringVar(&flags.ProjectRef, "project-ref", "", "Project ref of the Supabase project.")
	keyFlags.VarP(&keyOutput, "output", "o", "Output format of key variables.")
	keyFlags.StringSliceVar(&override, "override-name", []string{}, "Override specific variable names.")
	genCmd.AddCommand(genKeysCmd)
	rootCmd.AddCommand(genCmd)
}
