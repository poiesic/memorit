package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestReembedCommandFlags(t *testing.T) {
	app := &cli.App{
		Name: "memorit",
		Commands: []*cli.Command{
			{
				Name:   "reembed",
				Usage:  "Reembed all chat records with new embeddings",
				Action: reembedCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "db",
						Aliases:  []string{"d"},
						Usage:    "Path to BadgerDB database directory",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "embedding-host",
						Usage: "Embedding service host URL",
						Value: "http://localhost:11434/v1",
					},
					&cli.StringFlag{
						Name:     "embedding-model",
						Usage:    "Embedding model name",
						Required: true,
					},
					&cli.IntFlag{
						Name:  "batch-size",
						Usage: "Number of records to process in each batch",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "report-interval",
						Usage: "Report progress every N records",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "max-retries",
						Usage: "Maximum retry attempts for failed operations",
						Value: 3,
					},
				},
			},
		},
	}

	t.Run("embedding-model is required", func(t *testing.T) {
		args := []string{"memorit", "reembed", "--db", "/tmp/test"}
		err := app.Run(args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embedding-model")
	})

	t.Run("embedding-host has default value", func(t *testing.T) {
		cmd := app.Commands[0]
		var hostFlag *cli.StringFlag
		for _, flag := range cmd.Flags {
			if f, ok := flag.(*cli.StringFlag); ok && f.Name == "embedding-host" {
				hostFlag = f
				break
			}
		}
		require.NotNil(t, hostFlag)
		assert.Equal(t, "http://localhost:11434/v1", hostFlag.Value)
	})

	t.Run("embedding-model has no default value", func(t *testing.T) {
		cmd := app.Commands[0]
		var modelFlag *cli.StringFlag
		for _, flag := range cmd.Flags {
			if f, ok := flag.(*cli.StringFlag); ok && f.Name == "embedding-model" {
				modelFlag = f
				break
			}
		}
		require.NotNil(t, modelFlag)
		assert.Empty(t, modelFlag.Value)
		assert.True(t, modelFlag.Required)
	})

	t.Run("embedding-host has no EnvVars", func(t *testing.T) {
		cmd := app.Commands[0]
		var hostFlag *cli.StringFlag
		for _, flag := range cmd.Flags {
			if f, ok := flag.(*cli.StringFlag); ok && f.Name == "embedding-host" {
				hostFlag = f
				break
			}
		}
		require.NotNil(t, hostFlag)
		assert.Empty(t, hostFlag.EnvVars)
	})

	t.Run("embedding-model has no EnvVars", func(t *testing.T) {
		cmd := app.Commands[0]
		var modelFlag *cli.StringFlag
		for _, flag := range cmd.Flags {
			if f, ok := flag.(*cli.StringFlag); ok && f.Name == "embedding-model" {
				modelFlag = f
				break
			}
		}
		require.NotNil(t, modelFlag)
		assert.Empty(t, modelFlag.EnvVars)
	})

	t.Run("batch-size has default value of 100", func(t *testing.T) {
		cmd := app.Commands[0]
		var batchFlag *cli.IntFlag
		for _, flag := range cmd.Flags {
			if f, ok := flag.(*cli.IntFlag); ok && f.Name == "batch-size" {
				batchFlag = f
				break
			}
		}
		require.NotNil(t, batchFlag)
		assert.Equal(t, 100, batchFlag.Value)
	})

	t.Run("report-interval has default value of 100", func(t *testing.T) {
		cmd := app.Commands[0]
		var reportFlag *cli.IntFlag
		for _, flag := range cmd.Flags {
			if f, ok := flag.(*cli.IntFlag); ok && f.Name == "report-interval" {
				reportFlag = f
				break
			}
		}
		require.NotNil(t, reportFlag)
		assert.Equal(t, 100, reportFlag.Value)
	})

	t.Run("max-retries has default value of 3", func(t *testing.T) {
		cmd := app.Commands[0]
		var retriesFlag *cli.IntFlag
		for _, flag := range cmd.Flags {
			if f, ok := flag.(*cli.IntFlag); ok && f.Name == "max-retries" {
				retriesFlag = f
				break
			}
		}
		require.NotNil(t, retriesFlag)
		assert.Equal(t, 3, retriesFlag.Value)
	})
}

func TestReembedCommandValidation(t *testing.T) {
	// Set up a test environment that won't trigger database operations
	// by testing validation failures early
	app := &cli.App{
		Name: "memorit",
		Commands: []*cli.Command{
			{
				Name:   "reembed",
				Usage:  "Reembed all chat records with new embeddings",
				Action: reembedCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "db",
						Aliases:  []string{"d"},
						Usage:    "Path to BadgerDB database directory",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "embedding-host",
						Usage: "Embedding service host URL",
						Value: "http://localhost:11434/v1",
					},
					&cli.StringFlag{
						Name:     "embedding-model",
						Usage:    "Embedding model name",
						Required: true,
					},
					&cli.IntFlag{
						Name:  "batch-size",
						Usage: "Number of records to process in each batch",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "report-interval",
						Usage: "Report progress every N records",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "max-retries",
						Usage: "Maximum retry attempts for failed operations",
						Value: 3,
					},
				},
			},
		},
	}

	t.Run("missing db flag fails", func(t *testing.T) {
		args := []string{"memorit", "reembed", "--embedding-model", "test-model"}
		err := app.Run(args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db")
	})

	t.Run("missing embedding-model flag fails", func(t *testing.T) {
		args := []string{"memorit", "reembed", "--db", "/tmp/test"}
		err := app.Run(args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embedding-model")
	})
}

func TestSetupLogger(t *testing.T) {
	t.Run("valid log levels", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected slog.Level
		}{
			{"debug", slog.LevelDebug},
			{"info", slog.LevelInfo},
			{"warn", slog.LevelWarn},
			{"error", slog.LevelError},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				app := &cli.App{
					Name: "test",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "log-level",
							Value: tc.input,
						},
					},
					Before: setupLogger,
					Action: func(c *cli.Context) error {
						// Verify the logger was set up correctly by checking the default logger
						// This is a bit indirect but slog doesn't expose the level directly
						return nil
					},
				}

				err := app.Run([]string{"test", "--log-level", tc.input})
				require.NoError(t, err)
			})
		}
	})

	t.Run("case insensitive log levels", func(t *testing.T) {
		testCases := []string{
			"DEBUG",
			"Info",
			"WaRn",
			"ERROR",
		}

		for _, tc := range testCases {
			t.Run(tc, func(t *testing.T) {
				app := &cli.App{
					Name: "test",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "log-level",
							Value: "info",
						},
					},
					Before: setupLogger,
					Action: func(c *cli.Context) error {
						return nil
					},
				}

				err := app.Run([]string{"test", "--log-level", tc})
				require.NoError(t, err)
			})
		}
	})

	t.Run("invalid log level returns error", func(t *testing.T) {
		app := &cli.App{
			Name: "test",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "log-level",
					Value: "info",
				},
			},
			Before: setupLogger,
			Action: func(c *cli.Context) error {
				return nil
			},
		}

		err := app.Run([]string{"test", "--log-level", "invalid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid log level")
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("default log level is info", func(t *testing.T) {
		app := &cli.App{
			Name: "test",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "log-level",
					Value: "info",
				},
			},
			Before: setupLogger,
			Action: func(c *cli.Context) error {
				// Verify default is used when flag not specified
				level := c.String("log-level")
				assert.Equal(t, "info", level)
				return nil
			},
		}

		err := app.Run([]string{"test"})
		require.NoError(t, err)
	})

	t.Run("log-level flag has alias -l", func(t *testing.T) {
		app := &cli.App{
			Name: "test",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "log-level",
					Aliases: []string{"l"},
					Value:   "info",
				},
			},
			Before: setupLogger,
			Action: func(c *cli.Context) error {
				level := c.String("log-level")
				assert.Equal(t, "debug", level)
				return nil
			},
		}

		err := app.Run([]string{"test", "-l", "debug"})
		require.NoError(t, err)
	})
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}
