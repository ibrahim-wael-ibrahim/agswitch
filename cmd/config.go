package cmd
import("fmt";"strings";"github.com/spf13/cobra")
func newConfigCommand(d *dependencies)*cobra.Command{return &cobra.Command{Use:"config",Short:"Display the resolved configuration",Args:cobra.NoArgs,RunE:func(c *cobra.Command,_ []string)error{x:=d.config;_,e:=fmt.Fprintf(c.OutOrStdout(),"base_dir=%s\nstate_dir=%s\napp_path=%s\nlog_path=%s\nquit_command=%s\ngraceful_timeout=%s\nforce_kill=%t\n",x.BaseDir,x.StateDir,x.AppPath,x.LogPath,strings.Join(x.QuitCommand," "),x.GracefulTimeout,x.ForceKill);return e}}}
