package tui
import("context";tea "charm.land/bubbletea/v2";"github.com/ibrahim-wael/agswitch/internal/account";"github.com/ibrahim-wael/agswitch/internal/switcher")
type Backend interface{List(context.Context)([]account.Account,error);Use(context.Context,string,switcher.Options)error}
type Model struct{Context context.Context;Backend Backend;Accounts []account.Account;Selected int;Status string;Width,Height int;Stay,Switching bool}
func New(ctx context.Context,b Backend,a []account.Account,stay bool)Model{return Model{Context:ctx,Backend:b,Accounts:append([]account.Account(nil),a...),Status:"Ready",Stay:stay}}
func Run(ctx context.Context,b Backend,stay bool)error{a,e:=b.List(ctx);if e!=nil{return e};p:=tea.NewProgram(New(ctx,b,a,stay));_,e=p.Run();return e}
func(m Model)Init()tea.Cmd{return nil};type switchResultMsg struct{profile string;err error};func(m Model)switchCommand(p string)tea.Cmd{return func()tea.Msg{return switchResultMsg{p,m.Backend.Use(m.Context,p,switcher.Options{LaunchMode:switcher.AlwaysLaunch})}}}
