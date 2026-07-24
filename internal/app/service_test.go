package app

import("context";"errors";"testing";"github.com/ibrahim-wael/agswitch/internal/account";"github.com/ibrahim-wael/agswitch/internal/credentials")
type activeMemory struct{value credentials.Credential};func(m activeMemory)Load(context.Context)(credentials.Credential,error){return m.value,nil}
type profilesMemory map[string]credentials.Credential
func(m profilesMemory)Load(_ context.Context,n string)(credentials.Credential,error){v,ok:=m[n];if !ok{return credentials.Credential{},credentials.ErrNotFound};return v,nil}
func(m profilesMemory)Save(_ context.Context,n string,v credentials.Credential)error{m[n]=v;return nil};func(m profilesMemory)Delete(_ context.Context,n string)error{if _,ok:=m[n];!ok{return credentials.ErrNotFound};delete(m,n);return nil}
type accountsMemory map[string]account.Account
func(m accountsMemory)List(context.Context)([]account.Account,error){o:=[]account.Account{};for _,v:=range m{o=append(o,v)};return o,nil};func(m accountsMemory)Get(_ context.Context,id string)(account.Account,error){v,ok:=m[id];if !ok{return account.Account{},account.ErrNotFound};return v,nil};func(m accountsMemory)Save(_ context.Context,v account.Account)error{m[v.ID]=v;return nil};func(m accountsMemory)Delete(_ context.Context,id string)error{delete(m,id);return nil}
func TestSaveAndDetectCurrent(t *testing.T){c:=credentials.New([]byte(`{"email":"person@example.com"}`));p:=profilesMemory{};a:=accountsMemory{};s:=Service{Active:activeMemory{c},Profiles:p,Accounts:a};if err:=s.Save(context.Background(),"work",false);err!=nil{t.Fatal(err)};cur,ok,err:=s.Current(context.Background());if err!=nil||!ok||cur.ID!="work"{t.Fatalf("%#v %v %v",cur,ok,err)};if err=s.Save(context.Background(),"work",false);!errors.Is(err,ErrProfileExists){t.Fatal(err)}}
