package bflags

import (
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/eluv-io/errors-go"
)

// ViperOpts has options to bind with viper.
// note: the default config path must be set on the initialised struct that has the config binding
type ViperOpts struct {
	Viper      *viper.Viper          // viper instance or nil (one is created)
	Loaded     bool                  // true if viper was already loaded before binding
	ConfigFlag string                // name of the flag defining the config path
	viperFlags map[string]*flag.Flag // map viper key to flags
}

func NewViperOpts(v *viper.Viper, loaded bool, configFlag string) *ViperOpts {
	return &ViperOpts{
		Viper:      v,
		Loaded:     loaded,
		ConfigFlag: configFlag,
		viperFlags: make(map[string]*flag.Flag),
	}
}

func (v *ViperOpts) validate() error {
	e := errors.Template("validate", errors.K.Invalid)
	if v.Viper == nil {
		if v.Loaded {
			return e("reason", "a nil *viper.Viper cannot have been loaded:")
		}
		v.Viper = viper.New()
	}
	if !v.Loaded && v.ConfigFlag == "" {
		return e("reason", "viper will never be loaded: not loaded and no config flag")
	}
	return nil
}
