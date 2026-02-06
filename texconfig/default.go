package texconfig

import "sync"

var (
	defaultTexConvertOnce sync.Once        // Ensures the default config is initialized only once.
	defaultTexConvertMu   sync.RWMutex     // Protects access to the default config.
	defaultTexConvertCfgv TexConvertConfig // The default config.
	defaultTexConvertErr  error            // The error if the default config fails to load.
)

// initDefaultTexConvert initializes the default config from built-in values.
func initDefaultTexConvert() {
	defaultTexConvertMu.Lock()
	defer defaultTexConvertMu.Unlock()

	defaultTexConvertCfgv = defaultTexConvertConfig()
	defaultTexConvertErr = nil
}

// DefaultTexConvertConfig returns the current default config.
// The returned value is a copy and can be modified safely.
func DefaultTexConvertConfig() (TexConvertConfig, error) {
	defaultTexConvertOnce.Do(initDefaultTexConvert)
	defaultTexConvertMu.RLock()
	defer defaultTexConvertMu.RUnlock()

	return defaultTexConvertCfgv.Clone(), defaultTexConvertErr
}

// SetDefaultTexConvertConfig replaces the global default config.
// This is intended for callers that want to load a custom config once per process.
func SetDefaultTexConvertConfig(cfg TexConvertConfig) {
	defaultTexConvertOnce.Do(initDefaultTexConvert)
	defaultTexConvertMu.Lock()
	defaultTexConvertCfgv = cfg.Clone()
	defaultTexConvertErr = nil
	defaultTexConvertMu.Unlock()
}

// LoadDefaultTexConvertConfig loads a TexConvert.cfg and sets it as default.
func LoadDefaultTexConvertConfig(path string) error {
	cfg, err := LoadTexConvertConfig(path)
	if err != nil {
		return err
	}

	SetDefaultTexConvertConfig(cfg)

	return nil
}
