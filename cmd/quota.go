package cmd
import("fmt";"github.com/spf13/cobra")
func newQuotaCommand()*cobra.Command{return &cobra.Command{Use:"quota [profile]",Short:"Show quota information",Args:cobra.MaximumNArgs(1),RunE:func(c *cobra.Command,_ []string)error{fmt.Fprintln(c.ErrOrStderr(),"Live quota is not enabled yet because it depends on an unstable internal Google API.");return fmt.Errorf("quota provider unavailable")}}}
