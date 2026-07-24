package cmd
import("fmt";"github.com/ibrahim-wael/agswitch/internal/doctor";"github.com/spf13/cobra")
func newDoctorCommand(d *dependencies)*cobra.Command{return &cobra.Command{Use:"doctor",Short:"Inspect environment readiness",Args:cobra.NoArgs,RunE:func(c *cobra.Command,_ []string)error{x:=d.doctor.Run(c.Context());for _,v:=range x{fmt.Fprintf(c.OutOrStdout(),"[%s] %s: %s\n",v.Status,v.Name,v.Details)};if doctor.HasFailures(x){return fmt.Errorf("doctor found blocking problems")};return nil}}}
