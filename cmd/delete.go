package cmd
import("fmt";"github.com/spf13/cobra")
func newDeleteCommand(d *dependencies)*cobra.Command{var f bool;c:=&cobra.Command{Use:"delete <profile>",Aliases:[]string{"rm"},Short:"Delete a saved profile",Args:cobra.ExactArgs(1),RunE:func(c *cobra.Command,a []string)error{if e:=d.app.Delete(c.Context(),a[0],f);e!=nil{return e};_,e:=fmt.Fprintf(c.OutOrStdout(),"Deleted: %s\n",a[0]);return e}};c.Flags().BoolVarP(&f,"force","f",false,"allow deleting active profile");return c}
