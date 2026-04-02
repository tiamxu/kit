package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

type Tool interface {
	Name() string
	Description() string
	Commands() []*Command
	Flags() []Flag
}

type AppConfig struct {
	Name        string
	Description string
	Version     string
	PreRun      func(ctx *Context) error
}

type App struct {
	root          *cobra.Command
	tools         map[string]Tool
	cfg           AppConfig
	sharedContext *Context
}

func NewApp(cfg AppConfig) *App {
	a := &App{
		cfg:           cfg,
		tools:         make(map[string]Tool),
		sharedContext: &Context{Data: make(map[string]interface{})},
	}

	a.root = &cobra.Command{
		Use:   cfg.Name,
		Short: cfg.Description,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			a.sharedContext.Command = cmd
			if cfg.PreRun != nil {
				return cfg.PreRun(a.sharedContext)
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	a.root.PersistentFlags().StringP("config", "c", "", "config file path")
	a.root.PersistentFlags().Bool("debug", false, "debug mode")

	a.root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s v%s\n", cfg.Name, cfg.Version)
		},
	})

	return a
}

func (a *App) RegisterTool(tools ...Tool) *App {
	for _, t := range tools {
		name := t.Name()
		if _, exists := a.tools[name]; exists {
			continue
		}
		a.tools[name] = t

		cmd := &cobra.Command{
			Use:   name,
			Short: t.Description(),
		}

		for _, f := range t.Flags() {
			f.Apply(cmd)
		}

		for _, c := range t.Commands() {
			cmd.AddCommand(c.build(a))
		}

		a.root.AddCommand(cmd)
	}
	return a
}

func (a *App) Run() error {
	return a.root.Execute()
}

type Context struct {
	*cobra.Command
	Config interface{}
	Data   map[string]interface{}
	Args   []string
}

func (c *Context) String(name string) string {
	v, _ := c.Command.Flags().GetString(name)
	return v
}

func (c *Context) Int(name string) int {
	v, _ := c.Command.Flags().GetInt(name)
	return v
}

func (c *Context) Bool(name string) bool {
	v, _ := c.Command.Flags().GetBool(name)
	return v
}

func (c *Context) Duration(name string) time.Duration {
	v, _ := c.Command.Flags().GetDuration(name)
	return v
}

func (c *Context) StringSlice(name string) []string {
	v, _ := c.Command.Flags().GetStringSlice(name)
	return v
}

func (c *Context) StringMap(name string) map[string]string {
	v, _ := c.Command.Flags().GetStringToString(name)
	return v
}

func (c *Context) Set(key string, val interface{}) {
	c.Data[key] = val
}

func (c *Context) Get(key string) interface{} {
	return c.Data[key]
}

type Command struct {
	name        string
	usage       string
	description string
	flags       []Flag
	run         func(ctx *Context) error
	subCommands []*Command
	hidden      bool
}

func NewCommand(name string) *Command {
	return &Command{name: name}
}

func (c *Command) SetUsage(usage string) *Command {
	c.usage = usage
	return c
}

func (c *Command) SetDescription(desc string) *Command {
	c.description = desc
	return c
}

func (c *Command) AddFlags(flags ...Flag) *Command {
	c.flags = append(c.flags, flags...)
	return c
}

func (c *Command) AddCommand(cmds ...*Command) *Command {
	c.subCommands = append(c.subCommands, cmds...)
	return c
}

func (c *Command) SetRun(fn func(ctx *Context) error) *Command {
	c.run = fn
	return c
}

func (c *Command) Hidden() *Command {
	c.hidden = true
	return c
}

func (c *Command) build(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:    c.name,
		Short:  c.description,
		Hidden: c.hidden,
	}

	if c.usage != "" {
		cmd.Use = c.usage
	}

	for _, f := range c.flags {
		f.Apply(cmd)
	}

	if c.run != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			ctx := app.sharedContext
			ctx.Command = cmd
			ctx.Args = args
			return c.run(ctx)
		}
	}

	for _, sub := range c.subCommands {
		cmd.AddCommand(sub.build(app))
	}

	return cmd
}

type Flag interface {
	Apply(cmd *cobra.Command)
	markRequired(cmd *cobra.Command)
}

type stringFlag struct {
	name       string
	short      string
	defaultVal string
	usage      string
}

func StringFlag(name, short, defaultVal, usage string) Flag {
	return &stringFlag{name: name, short: short, defaultVal: defaultVal, usage: usage}
}

func (f *stringFlag) Apply(cmd *cobra.Command) {
	if f.short != "" {
		cmd.Flags().StringP(f.name, f.short, f.defaultVal, f.usage)
	} else {
		cmd.Flags().String(f.name, f.defaultVal, f.usage)
	}
}

func (f *stringFlag) markRequired(cmd *cobra.Command) {
	_ = cmd.MarkFlagRequired(f.name)
}

type intFlag struct {
	name       string
	short      string
	defaultVal int
	usage      string
}

func IntFlag(name, short string, defaultVal int, usage string) Flag {
	return &intFlag{name: name, short: short, defaultVal: defaultVal, usage: usage}
}

func (f *intFlag) Apply(cmd *cobra.Command) {
	if f.short != "" {
		cmd.Flags().IntP(f.name, f.short, f.defaultVal, f.usage)
	} else {
		cmd.Flags().Int(f.name, f.defaultVal, f.usage)
	}
}

func (f *intFlag) markRequired(cmd *cobra.Command) {
	_ = cmd.MarkFlagRequired(f.name)
}

type boolFlag struct {
	name       string
	short      string
	defaultVal bool
	usage      string
}

func BoolFlag(name, short string, defaultVal bool, usage string) Flag {
	return &boolFlag{name: name, short: short, defaultVal: defaultVal, usage: usage}
}

func (f *boolFlag) Apply(cmd *cobra.Command) {
	if f.short != "" {
		cmd.Flags().BoolP(f.name, f.short, f.defaultVal, f.usage)
	} else {
		cmd.Flags().Bool(f.name, f.defaultVal, f.usage)
	}
}

func (f *boolFlag) markRequired(cmd *cobra.Command) {
	_ = cmd.MarkFlagRequired(f.name)
}

type durationFlag struct {
	name       string
	short      string
	defaultVal time.Duration
	usage      string
}

func DurationFlag(name, short string, defaultVal time.Duration, usage string) Flag {
	return &durationFlag{name: name, short: short, defaultVal: defaultVal, usage: usage}
}

func (f *durationFlag) Apply(cmd *cobra.Command) {
	if f.short != "" {
		cmd.Flags().DurationP(f.name, f.short, f.defaultVal, f.usage)
	} else {
		cmd.Flags().Duration(f.name, f.defaultVal, f.usage)
	}
}

func (f *durationFlag) markRequired(cmd *cobra.Command) {
	_ = cmd.MarkFlagRequired(f.name)
}

type stringSliceFlag struct {
	name       string
	short      string
	defaultVal []string
	usage      string
}

func StringSliceFlag(name, short string, defaultVal []string, usage string) Flag {
	return &stringSliceFlag{name: name, short: short, defaultVal: defaultVal, usage: usage}
}

func (f *stringSliceFlag) Apply(cmd *cobra.Command) {
	if f.short != "" {
		cmd.Flags().StringSliceP(f.name, f.short, f.defaultVal, f.usage)
	} else {
		cmd.Flags().StringSlice(f.name, f.defaultVal, f.usage)
	}
}

func (f *stringSliceFlag) markRequired(cmd *cobra.Command) {
	_ = cmd.MarkFlagRequired(f.name)
}

type requiredFlag struct {
	inner Flag
}

func RequiredFlag(flag Flag) Flag {
	return &requiredFlag{inner: flag}
}

func (f *requiredFlag) Apply(cmd *cobra.Command) {
	f.inner.Apply(cmd)
	f.inner.markRequired(cmd)
}

func (f *requiredFlag) markRequired(cmd *cobra.Command) {
	f.inner.markRequired(cmd)
}
