// +build !windows

package shadow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/foxcpp/maddy/auth/external"
	"github.com/foxcpp/maddy/config"
	"github.com/foxcpp/maddy/log"
	"github.com/foxcpp/maddy/module"
)

type Auth struct {
	instName   string
	useHelper  bool
	helperPath string

	Log log.Logger
}

func New(modName, instName string, _ []string) (module.Module, error) {
	return &Auth{
		instName: instName,
		Log:      log.Logger{Name: modName},
	}, nil
}

func (a *Auth) Name() string {
	return "shadow"
}

func (a *Auth) InstanceName() string {
	return a.instName
}

func (a *Auth) Init(cfg *config.Map) error {
	cfg.Bool("debug", true, &a.Log.Debug)
	cfg.Bool("use_helper", false, &a.useHelper)
	if _, err := cfg.Process(); err != nil {
		return err
	}

	if a.useHelper {
		a.helperPath = filepath.Join(config.LibexecDirectory(cfg.Globals), "maddy-shadow-helper")
		if _, err := os.Stat(a.helperPath); err != nil {
			return fmt.Errorf("shadow: no helper binary (maddy-shadow-helper) found in %s", config.LibexecDirectory(cfg.Globals))
		}
	} else {
		f, err := os.Open("/etc/shadow")
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("shadow: can't read /etc/shadow due to permission error, use helper binary or run maddy as a privileged user")
			}
			return fmt.Errorf("shadow: can't read /etc/shadow: %v", err)
		}
		f.Close()
	}

	return nil
}

func (a *Auth) CheckPlain(username, password string) bool {
	if a.useHelper {
		return external.AuthUsingHelper(a.Log, a.helperPath, username, password)
	}

	ent, err := Lookup(username)
	if err != nil {
		if err != ErrNoSuchUser {
			a.Log.Printf("%v, username = %s", err, username)
		}
		return false
	}

	if !ent.IsAccountValid() {
		a.Log.Printf("account is expired, username = %s", username)
		return false
	}

	if !ent.IsPasswordValid() {
		a.Log.Printf("password is expired, username = %s", username)
		return false
	}

	if err := ent.VerifyPassword(password); err != nil {
		if err != ErrWrongPassword {
			a.Log.Printf("%v", err)
		}
		a.Log.Debugf("password verification failed, username = %s", username)
		return false
	}

	return true
}

func init() {
	module.Register("shadow", New)
}
