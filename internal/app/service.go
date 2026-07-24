package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	"github.com/ibrahim-wael/agswitch/internal/credentials"
	"github.com/ibrahim-wael/agswitch/internal/profile"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
)

var (
	ErrProfileExists = errors.New("profile already exists")
	ErrActiveProfile = errors.New("cannot delete the active profile without force")
)

type ActiveStore interface { Load(ctx context.Context) (credentials.Credential, error) }
type ProfileStore interface { Load(context.Context,string)(credentials.Credential,error); Save(context.Context,string,credentials.Credential) error; Delete(context.Context,string) error }
type AccountRepository interface { List(context.Context)([]account.Account,error); Get(context.Context,string)(account.Account,error); Save(context.Context,account.Account) error; Delete(context.Context,string) error }
type Locker interface { Lock(context.Context)(func() error,error) }
type Switcher interface { SwitchWithOptions(context.Context,string,switcher.Options) error }

type Service struct { Active ActiveStore; Profiles ProfileStore; Accounts AccountRepository; Switcher Switcher; Locker Locker; Now func() time.Time }

func (s *Service) Save(ctx context.Context,name string,force bool) error { if s.Active==nil{return errors.New("active credential store is not configured")}; c,err:=s.Active.Load(ctx); if err!=nil{return fmt.Errorf("read active Antigravity credential: %w",err)}; c,err=credentials.Parse(c.Raw); if err!=nil{return fmt.Errorf("parse active Antigravity credential: %w",err)}; c.Source="active-keyring"; return s.saveCredential(ctx,name,c,force) }
func (s *Service) Import(ctx context.Context,name string,raw []byte,force bool) error { c,err:=credentials.Parse(raw); if err!=nil{return err}; c.Source="legacy-file"; return s.saveCredential(ctx,name,c,force) }
func (s *Service) List(ctx context.Context)([]account.Account,error){ if s.Accounts==nil{return nil,errors.New("account repository is not configured")}; items,err:=s.Accounts.List(ctx); if err!=nil{return nil,err}; fp:=""; if s.Active!=nil { c,e:=s.Active.Load(ctx); if e==nil{fp=c.Fingerprint}else if !errors.Is(e,credentials.ErrNotFound){return nil,fmt.Errorf("read active credential: %w",e)} }; for i:=range items{items[i].Active=fp!=""&&items[i].CredentialFingerprint==fp}; sort.Slice(items,func(i,j int)bool{return items[i].ID<items[j].ID}); return items,nil }
func (s *Service) Current(ctx context.Context)(account.Account,bool,error){items,err:=s.List(ctx);if err!=nil{return account.Account{},false,err};for _,v:=range items{if v.Active{return v,true,nil}};return account.Account{},false,nil}
func (s *Service) Delete(ctx context.Context,name string,force bool)(err error){if err:=profile.Validate(name);err!=nil{return err};if s.Profiles==nil||s.Accounts==nil{return errors.New("profile storage is not configured")};return s.withLock(ctx,func()error{cur,found,e:=s.Current(ctx);if e!=nil{return e};if found&&cur.ID==name&&!force{return ErrActiveProfile};backup,e:=s.Profiles.Load(ctx,name);if e!=nil{return e};if e=s.Profiles.Delete(ctx,name);e!=nil{return e};if e=s.Accounts.Delete(ctx,name);e!=nil{if r:=s.Profiles.Save(ctx,name,backup);r!=nil{return errors.Join(e,fmt.Errorf("restore profile after metadata failure: %w",r))};return e};return nil})}
func (s *Service) Use(ctx context.Context,name string,opts switcher.Options) error {if err:=profile.Validate(name);err!=nil{return err};if s.Switcher==nil{return errors.New("switcher is not configured")};if err:=s.Switcher.SwitchWithOptions(ctx,name,opts);err!=nil{return err};if s.Accounts!=nil{item,err:=s.Accounts.Get(ctx,name);if err==nil{item.LastUsedAt=s.now();if e:=s.Accounts.Save(ctx,item);e!=nil{return fmt.Errorf("profile switched but metadata update failed: %w",e)}}};return nil}

type MigrationResult struct{Profile,Path,Status string;Err error}
func (s *Service) Migrate(ctx context.Context,dir string,force,deleteSource bool)([]MigrationResult,error){entries,err:=os.ReadDir(dir);if errors.Is(err,os.ErrNotExist){return nil,nil};if err!=nil{return nil,err};out:=[]MigrationResult{};for _,entry:=range entries{if ctx.Err()!=nil{return out,ctx.Err()};name:=entry.Name();if entry.IsDir()||!strings.HasSuffix(name,".json")||name=="accounts.json"||name=="config.json"{continue};path:=filepath.Join(dir,name);profileName:=strings.TrimSuffix(name,".json");r:=MigrationResult{Profile:profileName,Path:path};info,e:=os.Lstat(path);if e!=nil||info.Mode()&os.ModeSymlink!=0||!info.Mode().IsRegular(){if e==nil{e=errors.New("profile source must be a regular non-symlink file")};r.Status,r.Err="failed",e;out=append(out,r);continue};if e=os.Chmod(path,0600);e!=nil{r.Status,r.Err="failed",e;out=append(out,r);continue};raw,e:=os.ReadFile(path);if e==nil{e=s.Import(ctx,profileName,raw,force)};if e!=nil{r.Status,r.Err="failed",e;out=append(out,r);continue};r.Status="migrated";if deleteSource{if e=os.Remove(path);e!=nil{r.Status,r.Err="migrated-source-kept",e}};out=append(out,r)};sort.Slice(out,func(i,j int)bool{return out[i].Profile<out[j].Profile});return out,nil}
func (s *Service) saveCredential(ctx context.Context,name string,c credentials.Credential,force bool) error {if err:=profile.Validate(name);err!=nil{return err};if s.Profiles==nil||s.Accounts==nil{return errors.New("profile storage is not configured")};return s.withLock(ctx,func()error{prev,e:=s.Profiles.Load(ctx,name);existed:=e==nil;if e!=nil&&!errors.Is(e,credentials.ErrNotFound){return e};if existed&&!force{return ErrProfileExists};if e=s.Profiles.Save(ctx,name,c);e!=nil{return e};v,e:=s.Profiles.Load(ctx,name);if e!=nil||v.Fingerprint!=c.Fingerprint{rb:=s.rollbackProfile(ctx,name,prev,existed);if e==nil{e=errors.New("profile verification fingerprint mismatch")};return errors.Join(e,rb)};item:=account.Account{ID:name,Email:c.Email,CredentialFingerprint:c.Fingerprint,QuotaEnabled:true};if old,g:=s.Accounts.Get(ctx,name);g==nil{item.CreatedAt=old.CreatedAt;item.Label=old.Label;item.LastUsedAt=old.LastUsedAt;item.QuotaEnabled=old.QuotaEnabled};if e=s.Accounts.Save(ctx,item);e!=nil{return errors.Join(e,s.rollbackProfile(ctx,name,prev,existed))};return nil})}
func (s *Service) rollbackProfile(ctx context.Context,name string,prev credentials.Credential,existed bool) error {if existed{return s.Profiles.Save(ctx,name,prev)};e:=s.Profiles.Delete(ctx,name);if errors.Is(e,credentials.ErrNotFound){return nil};return e}
func (s *Service) withLock(ctx context.Context,fn func()error)(err error){if s.Locker==nil{return fn()};unlock,err:=s.Locker.Lock(ctx);if err!=nil{return err};defer func(){if e:=unlock();err==nil&&e!=nil{err=e}}();return fn()}
func (s *Service) now() time.Time {if s.Now!=nil{return s.Now().UTC()};return time.Now().UTC()}
